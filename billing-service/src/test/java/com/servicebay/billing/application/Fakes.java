package com.servicebay.billing.application;

import com.servicebay.billing.domain.bill.Bill;
import com.servicebay.billing.domain.bill.BillLine;
import com.servicebay.billing.domain.bill.BillStatus;
import com.servicebay.billing.domain.booking.AppointmentRef;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.outbox.OutboxMessage;
import com.servicebay.billing.domain.payment.Payment;
import com.servicebay.billing.domain.payment.Refund;
import com.servicebay.billing.domain.ports.AppointmentRefRepository;
import com.servicebay.billing.domain.ports.BillRepository;
import com.servicebay.billing.domain.ports.ConflictException;
import com.servicebay.billing.domain.ports.Directories;
import com.servicebay.billing.domain.ports.OutboxRepository;
import com.servicebay.billing.domain.ports.PaymentProvider;
import com.servicebay.billing.domain.ports.PaymentProviderException;
import com.servicebay.billing.domain.ports.PaymentRepository;
import com.servicebay.billing.domain.ports.ProcessedEventRepository;
import com.servicebay.billing.domain.ports.RefundRepository;
import com.servicebay.billing.domain.ports.StripeCustomerRepository;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Set;
import java.util.concurrent.atomic.AtomicLong;

/** In-memory versions of every port, so use cases run as a whole without a database or Stripe. */
final class Fakes {

    private Fakes() {
    }

    static final class Bills implements BillRepository {
        final Map<Long, Bill> rows = new LinkedHashMap<>();
        private final AtomicLong ids = new AtomicLong();
        private final AtomicLong lineIds = new AtomicLong();

        @Override
        public Bill save(Bill bill) {
            boolean otherLive = rows.values().stream().anyMatch(b -> !b.id().equals(bill.id())
                    && b.appointmentId() == bill.appointmentId() && b.status() != BillStatus.VOID);
            if (otherLive && bill.status() != BillStatus.VOID) {
                throw new ConflictException("live bill exists for appointment " + bill.appointmentId());
            }
            if (bill.id() != null && rows.get(bill.id()).version() != bill.version()) {
                throw new ConflictException("stale version");
            }
            long id = bill.id() == null ? ids.incrementAndGet() : bill.id();
            List<BillLine> lines = bill.lines().stream()
                    .map(l -> l.id() != null ? l : new BillLine(lineIds.incrementAndGet(), l.kind(), l.sourceId(),
                            l.name(), l.quantity(), l.unitPrice()))
                    .toList();
            Bill saved = new Bill(id, bill.appointmentId(), bill.customerId(), bill.vehicleId(), bill.currency(),
                    bill.warrantyFee(), bill.serviceFee(), bill.materialCost(), bill.status(), bill.finalizedAt(),
                    lines, bill.id() == null ? 0 : bill.version() + 1);
            rows.put(id, saved);
            return saved;
        }

        @Override
        public Optional<Bill> findById(long id) {
            return Optional.ofNullable(rows.get(id));
        }

        @Override
        public Optional<Bill> findByIdForUpdate(long id) {
            return findById(id);
        }

        @Override
        public Optional<Bill> findLiveByAppointment(long appointmentId) {
            return rows.values().stream()
                    .filter(b -> b.appointmentId() == appointmentId && b.status() != BillStatus.VOID).findFirst();
        }

        @Override
        public List<Bill> findByCustomer(long customerId) {
            return findByCustomer(customerId, null, Integer.MAX_VALUE);
        }

        @Override
        public List<Bill> findByCustomer(long customerId, Long beforeId, int limit) {
            return rows.values().stream()
                    .filter(b -> b.customerId() == customerId && (beforeId == null || b.id() < beforeId))
                    .sorted(Comparator.comparing(Bill::id).reversed()).limit(limit).toList();
        }
    }

    static final class Payments implements PaymentRepository {
        final Map<Long, Payment> rows = new LinkedHashMap<>();
        private final AtomicLong ids = new AtomicLong();

        @Override
        public Payment save(Payment p) {
            boolean dupKey = rows.values().stream()
                    .anyMatch(o -> !o.id().equals(p.id()) && o.idempotencyKey().equals(p.idempotencyKey()));
            if (dupKey) {
                throw new ConflictException("duplicate idempotency key");
            }
            long id = p.id() == null ? ids.incrementAndGet() : p.id();
            Payment saved = new Payment(id, p.billId(), p.type(), p.amount(), p.status(), p.stripePaymentIntentId(),
                    p.idempotencyKey(), p.failureCode(), p.failureMessage(), p.cardBrand(), p.cardLast4(),
                    p.succeededAt(), p.id() == null ? 0 : p.version() + 1);
            rows.put(id, saved);
            return saved;
        }

        @Override
        public Optional<Payment> findById(long id) {
            return Optional.ofNullable(rows.get(id));
        }

        @Override
        public Optional<Payment> findByIdempotencyKey(String key) {
            return rows.values().stream().filter(p -> p.idempotencyKey().equals(key)).findFirst();
        }

        @Override
        public Optional<Payment> findByStripePaymentIntentId(String id) {
            return rows.values().stream().filter(p -> id.equals(p.stripePaymentIntentId())).findFirst();
        }

        @Override
        public List<Payment> findByBill(long billId) {
            return rows.values().stream().filter(p -> p.billId() == billId).toList();
        }
    }

    static final class Refunds implements RefundRepository {
        @Override
        public Refund save(Refund refund) {
            throw new UnsupportedOperationException();
        }

        @Override
        public Optional<Refund> findByStripeRefundId(String id) {
            return Optional.empty();
        }

        @Override
        public List<Refund> findByPayment(long paymentId) {
            return List.of();
        }

        @Override
        public List<Refund> findByBill(long billId) {
            return List.of();
        }
    }

    static final class Appointments implements AppointmentRefRepository {
        final Map<Long, AppointmentRef> rows = new HashMap<>();

        @Override
        public void upsert(AppointmentRef ref) {
            rows.put(ref.appointmentId(), ref);
        }

        @Override
        public Optional<AppointmentRef> findById(long id) {
            return Optional.ofNullable(rows.get(id));
        }
    }

    static final class Processed implements ProcessedEventRepository {
        final Set<String> seen = new HashSet<>();

        @Override
        public boolean markProcessed(String source, String eventId) {
            return seen.add(source + "/" + eventId);
        }
    }

    static final class Outbox implements OutboxRepository {
        final List<OutboxMessage> messages = new ArrayList<>();

        @Override
        public void append(OutboxMessage message) {
            messages.add(message);
        }

        List<String> topics() {
            return messages.stream().map(OutboxMessage::topic).toList();
        }
    }

    static final class ProviderCustomers implements StripeCustomerRepository {
        final Map<Long, String> rows = new HashMap<>();

        @Override
        public Optional<String> findStripeCustomerId(long customerId) {
            return Optional.ofNullable(rows.get(customerId));
        }

        @Override
        public void save(long customerId, String id) {
            if (rows.putIfAbsent(customerId, id) != null) {
                throw new ConflictException("exists");
            }
        }
    }

    /** A provider that behaves like Stripe in test mode, including idempotency keys. */
    static final class Provider implements PaymentProvider {
        final Map<String, ProviderIntent> intents = new LinkedHashMap<>();
        final Map<String, String> byIdempotencyKey = new HashMap<>();
        final List<String> canceled = new ArrayList<>();
        int creates;
        boolean failCreate;

        @Override
        public String createCustomer(long customerId, String key) {
            return "cus_" + customerId;
        }

        @Override
        public ProviderIntent createIntent(Money amount, String customer, Map<String, String> metadata, String key) {
            if (failCreate) {
                throw new PaymentProviderException(PaymentProviderException.Kind.UNAVAILABLE, "stripe is down");
            }
            if (byIdempotencyKey.containsKey(key)) {
                return intents.get(byIdempotencyKey.get(key));
            }
            creates++;
            String id = "pi_" + creates;
            ProviderIntent intent = new ProviderIntent(id, id + "_secret", "requires_payment_method",
                    amount.amountMinor(), 0, null, null, null, null, Long.valueOf(metadata.get("payment_id")));
            intents.put(id, intent);
            byIdempotencyKey.put(key, id);
            return intent;
        }

        @Override
        public ProviderIntent retrieveIntent(String id) {
            return intents.get(id);
        }

        @Override
        public ProviderIntent cancelIntent(String id) {
            ProviderIntent i = intents.get(id);
            if (i.status().equals("succeeded") || i.status().equals("processing")) {
                throw new PaymentProviderException(PaymentProviderException.Kind.CONFLICT, "unexpected state");
            }
            canceled.add(id);
            return set(id, "canceled", 0, "canceled", "canceled: requested_by_customer", null, null);
        }

        @Override
        public ProviderIntent confirmTestIntent(String id, String method) {
            if (method.equals("pm_card_chargeDeclined")) {
                return set(id, "requires_payment_method", 0, "generic_decline", "Your card was declined.", null, null);
            }
            return succeed(id);
        }

        ProviderIntent succeed(String id) {
            return set(id, "succeeded", intents.get(id).amount(), null, null, "visa", "4242");
        }

        ProviderIntent set(String id, String status, long received, String code, String message, String brand,
                String last4) {
            ProviderIntent i = intents.get(id);
            ProviderIntent next = new ProviderIntent(id, i.clientSecret(), status, i.amount(), received, code, message,
                    brand, last4, i.paymentId());
            intents.put(id, next);
            return next;
        }

        @Override
        public boolean isTestMode() {
            return true;
        }

        @Override
        public ProviderEvent parseWebhook(String payload, String signatureHeader) {
            throw new UnsupportedOperationException();
        }
    }

    static final class Customers implements Directories.CustomerDirectory {
        final Map<String, Long> byToken = new HashMap<>();

        @Override
        public Optional<Long> currentCustomerId(String token) {
            return Optional.ofNullable(byToken.get(token));
        }
    }

    static final class Vehicles implements Directories.VehicleDirectory {
        Warranty answer = new Warranty(false, null);
        LocalDate askedFor;
        boolean down;

        @Override
        public Warranty warranty(long vehicleId, LocalDate asOf) {
            if (down) {
                throw new Directories.UpstreamException("vehicle-service unreachable", null);
            }
            askedFor = asOf;
            return answer;
        }
    }

    static final class Scheduler implements Directories.SchedulerDirectory {
        final Map<Long, AppointmentRef> mine = new HashMap<>();

        @Override
        public Optional<AppointmentRef> findMyAppointment(String token, long appointmentId) {
            return Optional.ofNullable(mine.get(appointmentId));
        }
    }
}
