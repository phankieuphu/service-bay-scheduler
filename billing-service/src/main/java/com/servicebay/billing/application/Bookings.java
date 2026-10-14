package com.servicebay.billing.application;

import com.servicebay.billing.domain.balance.Balance;
import com.servicebay.billing.domain.bill.Bill;
import com.servicebay.billing.domain.bill.BillLine;
import com.servicebay.billing.domain.bill.BillStatus;
import com.servicebay.billing.domain.bill.LineKind;
import com.servicebay.billing.domain.bill.Pricing;
import com.servicebay.billing.domain.booking.AppointmentRef;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.payment.Payment;
import com.servicebay.billing.domain.payment.PaymentStatus;
import com.servicebay.billing.domain.ports.AppointmentRefRepository;
import com.servicebay.billing.domain.ports.BillRepository;
import com.servicebay.billing.domain.ports.ConflictException;
import com.servicebay.billing.domain.ports.Directories.VehicleDirectory;
import com.servicebay.billing.domain.ports.Directories.VehicleDirectory.Warranty;
import com.servicebay.billing.domain.ports.OutboxRepository;
import com.servicebay.billing.domain.ports.PaymentProvider;
import com.servicebay.billing.domain.ports.PaymentProvider.ProviderIntent;
import com.servicebay.billing.domain.ports.PaymentProviderException;
import com.servicebay.billing.domain.ports.PaymentRepository;
import com.servicebay.billing.domain.ports.ProcessedEventRepository;
import com.servicebay.billing.domain.ports.RefundRepository;
import java.time.Clock;
import java.time.Instant;
import java.time.LocalDate;
import java.time.ZoneId;
import java.util.List;
import java.util.Optional;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionOperations;

/**
 * What billing does with scheduler's events (architecture-design §3d, §3b):
 * keep the booking snapshot that deposits are priced from, void an unpaid
 * bill when a booking is cancelled, and price the bill when the service is
 * completed (#85).
 *
 * <p>Every handler is idempotent: the event id is recorded in processed_event
 * in the same transaction as the change, and it's recorded last, so a
 * handler that fails half way (a provider call, a warranty lookup) is retried
 * in full.
 */
@Service
public class Bookings {

    private static final Logger log = LoggerFactory.getLogger(Bookings.class);

    private final AppointmentRefRepository appointments;
    private final BillRepository bills;
    private final PaymentRepository payments;
    private final RefundRepository refunds;
    private final ProcessedEventRepository processed;
    private final OutboxRepository outbox;
    private final VehicleDirectory vehicles;
    private final PaymentProvider provider;
    private final Payments paymentUseCases;
    private final TransactionOperations tx;
    private final BillingProperties properties;
    private final Clock clock;

    Bookings(AppointmentRefRepository appointments, BillRepository bills, PaymentRepository payments,
            RefundRepository refunds, ProcessedEventRepository processed, OutboxRepository outbox,
            VehicleDirectory vehicles, PaymentProvider provider, Payments paymentUseCases, TransactionOperations tx,
            BillingProperties properties, Clock clock) {
        this.appointments = appointments;
        this.bills = bills;
        this.payments = payments;
        this.refunds = refunds;
        this.processed = processed;
        this.outbox = outbox;
        this.vehicles = vehicles;
        this.provider = provider;
        this.paymentUseCases = paymentUseCases;
        this.tx = tx;
        this.properties = properties;
        this.clock = clock;
    }

    /** One performed service, as ServiceCompleted carries it. */
    public record CompletedService(long serviceId, String name, int quantity, long unitPriceMinor) {
    }

    /** ServiceCompleted (§3b plus the priced fields of §3e). */
    public record Completion(long appointmentId, long customerId, long vehicleId, long dealershipId,
            Instant completedAt, String currency, List<CompletedService> services) {
    }

    /** AppointmentBooked: remember the booking, so its deposit can be priced. */
    public void booked(String source, String eventId, AppointmentRef booking) {
        tx.executeWithoutResult(s -> {
            if (!processed.markProcessed(source, eventId)) {
                return;
            }
            // Per-key ordering means a later state can only be from an older
            // delivery of this booking's history; never move back to BOOKED.
            Optional<AppointmentRef> known = appointments.findById(booking.appointmentId());
            if (known.isEmpty() || known.get().status() == AppointmentRef.Status.BOOKED) {
                appointments.upsert(booking);
            }
        });
    }

    /**
     * AppointmentCancelled: open deposit intents are canceled at the provider
     * and an unpriced bill with nothing paid is voided. A paid deposit is left
     * on the OPEN bill: refunding it is the cancellation policy's call (#89).
     */
    public void cancelled(String source, String eventId, AppointmentRef booking) {
        Optional<Bill> bill = bills.findLiveByAppointment(booking.appointmentId());
        if (bill.isPresent() && bill.get().status() == BillStatus.OPEN) {
            for (Payment p : payments.findByBill(bill.get().id())) {
                if (p.status() == PaymentStatus.REQUIRES_PAYMENT && p.stripePaymentIntentId() != null) {
                    cancelIntent(p);
                }
            }
        }

        tx.executeWithoutResult(s -> {
            if (!processed.markProcessed(source, eventId)) {
                return;
            }
            appointments.upsert(appointments.findById(booking.appointmentId())
                    .map(ref -> ref.withStatus(AppointmentRef.Status.CANCELLED))
                    .orElse(booking.withStatus(AppointmentRef.Status.CANCELLED)));
            if (bill.isEmpty()) {
                return;
            }
            Bill locked = bills.findByIdForUpdate(bill.get().id()).orElseThrow();
            if (locked.status() != BillStatus.OPEN) {
                return;
            }
            List<Payment> all = payments.findByBill(locked.id());
            if (all.stream().anyMatch(p -> p.status().isCaptured() || p.status() == PaymentStatus.PROCESSING)) {
                log.warn("appointment {} was cancelled with a deposit paid or processing on bill {}; "
                        + "the bill stays OPEN until a refund decision (#89)", locked.appointmentId(), locked.id());
                return;
            }
            bills.save(locked.withStatus(BillStatus.VOID));
        });
    }

    private void cancelIntent(Payment payment) {
        ProviderIntent intent;
        try {
            intent = provider.cancelIntent(payment.stripePaymentIntentId());
        } catch (PaymentProviderException e) {
            if (e.kind() != PaymentProviderException.Kind.CONFLICT) {
                throw e; // retried by the consumer
            }
            intent = provider.retrieveIntent(payment.stripePaymentIntentId());
        }
        ProviderIntent seen = intent;
        tx.executeWithoutResult(s -> paymentUseCases.apply(payment.id(), seen, false));
    }

    /**
     * ServiceCompleted: prices the appointment's bill (creating it if no
     * deposit did). The warranty is looked up for the completion date before
     * the transaction; if vehicle-service can't answer, this throws and the
     * consumer retries. It never assumes the vehicle is in warranty.
     *
     * <p>Material costs arrive with MaterialsConsumed once dealership-service
     * exists; until then the bill is final on ServiceCompleted alone.
     */
    public void completed(String source, String eventId, Completion completion) {
        LocalDate serviceDate = completion.completedAt().atZone(ZoneId.of(properties.timeZone())).toLocalDate();
        Warranty warranty = vehicles.warranty(completion.vehicleId(), serviceDate);

        tx.executeWithoutResult(s -> {
            if (!processed.markProcessed(source, eventId)) {
                return;
            }
            String currency = completion.currency();
            Money estimate = completion.services().stream()
                    .map(svc -> Money.of(svc.unitPriceMinor(), currency).times(svc.quantity()))
                    .reduce(Money.zero(currency), Money::plus);
            appointments.upsert(appointments.findById(completion.appointmentId())
                    .map(ref -> ref.withStatus(AppointmentRef.Status.COMPLETED))
                    .orElse(new AppointmentRef(completion.appointmentId(), completion.customerId(),
                            completion.vehicleId(), completion.dealershipId(), estimate,
                            AppointmentRef.Status.COMPLETED)));

            Bill bill = bills.findLiveByAppointment(completion.appointmentId()).orElseGet(() -> bills.save(
                    Bill.open(completion.appointmentId(), completion.customerId(), completion.vehicleId(), currency)));
            Bill locked = bills.findByIdForUpdate(bill.id()).orElseThrow();
            if (locked.status() != BillStatus.OPEN) {
                log.info("bill {} for appointment {} is already {}; ServiceCompleted {} changes nothing",
                        locked.id(), locked.appointmentId(), locked.status(), eventId);
                return;
            }
            if (!locked.currency().equals(currency)) {
                // A deposit taken in another currency than the catalog's: a
                // configuration error no retry fixes, so it goes to the DLT.
                throw new ConflictException("bill " + locked.id() + " is in " + locked.currency()
                        + " but ServiceCompleted is in " + currency);
            }

            List<BillLine> serviceLines = completion.services().stream()
                    .map(svc -> new BillLine(null, LineKind.SERVICE, svc.serviceId(), svc.name(), svc.quantity(),
                            Money.of(svc.unitPriceMinor(), currency)))
                    .toList();
            Money fee = Money.ofMajor(properties.warrantyFee(), currency);
            List<BillLine> lines = Pricing.lines(serviceLines, List.of(), warranty.inWarranty(), warranty.endDate(), fee);

            Bill priced = locked.price(lines, Instant.now(clock));
            List<Payment> all = payments.findByBill(priced.id());
            Balance balance = Balance.of(priced, all, refunds.findByBill(priced.id()));
            Bill saved = bills.save(priced.withStatus(balance.statusFor(priced)));

            if (balance.due().isPositive()) {
                outbox.append(BillingEvents.billReady(saved, balance));
            }
            if (balance.refundDue().isPositive()) {
                log.warn("bill {} is priced below its deposit; {} is due back (#89)", saved.id(), balance.refundDue());
            }
            log.info("priced bill {} for appointment {}: {} (warranty {}), status {}", saved.id(),
                    saved.appointmentId(), saved.total(), warranty.inWarranty() ? "active" : "not active",
                    saved.status());
        });
    }
}
