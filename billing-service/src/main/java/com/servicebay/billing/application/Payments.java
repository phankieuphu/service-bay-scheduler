package com.servicebay.billing.application;

import com.servicebay.billing.domain.balance.Balance;
import com.servicebay.billing.domain.bill.Bill;
import com.servicebay.billing.domain.bill.BillStatus;
import com.servicebay.billing.domain.booking.AppointmentRef;
import com.servicebay.billing.domain.deposit.DepositPolicy;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.payment.Payment;
import com.servicebay.billing.domain.payment.PaymentStatus;
import com.servicebay.billing.domain.payment.PaymentType;
import com.servicebay.billing.domain.ports.AppointmentRefRepository;
import com.servicebay.billing.domain.ports.BillRepository;
import com.servicebay.billing.domain.ports.ConflictException;
import com.servicebay.billing.domain.ports.Directories.SchedulerDirectory;
import com.servicebay.billing.domain.ports.OutboxRepository;
import com.servicebay.billing.domain.ports.PaymentProvider;
import com.servicebay.billing.domain.ports.PaymentProvider.ProviderEvent;
import com.servicebay.billing.domain.ports.PaymentProvider.ProviderIntent;
import com.servicebay.billing.domain.ports.PaymentProviderException;
import com.servicebay.billing.domain.ports.PaymentRepository;
import com.servicebay.billing.domain.ports.ProcessedEventRepository;
import com.servicebay.billing.domain.ports.RefundRepository;
import com.servicebay.billing.domain.ports.StripeCustomerRepository;
import java.time.Clock;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionOperations;

/**
 * Taking money: a deposit at booking (#86), paying the balance (#88), and
 * applying what the provider says happened (#83: webhooks, re-reads, test
 * confirmations).
 *
 * <p>Every change to a bill's payments runs under the bill's row lock, so the
 * balance a payment is created for and the status it settles to can't
 * interleave. Provider calls run between transactions, never inside one.
 */
@Service
public class Payments {

    private static final Logger log = LoggerFactory.getLogger(Payments.class);

    /** processed_event source for webhooks. */
    static final String STRIPE_SOURCE = "stripe";

    private final BillRepository bills;
    private final PaymentRepository payments;
    private final RefundRepository refunds;
    private final AppointmentRefRepository appointments;
    private final StripeCustomerRepository providerCustomers;
    private final ProcessedEventRepository processed;
    private final OutboxRepository outbox;
    private final PaymentProvider provider;
    private final SchedulerDirectory scheduler;
    private final Access access;
    private final TransactionOperations tx;
    private final DepositPolicy depositPolicy;
    private final Clock clock;

    Payments(BillRepository bills, PaymentRepository payments, RefundRepository refunds,
            AppointmentRefRepository appointments, StripeCustomerRepository providerCustomers,
            ProcessedEventRepository processed, OutboxRepository outbox, PaymentProvider provider,
            SchedulerDirectory scheduler, Access access, TransactionOperations tx, DepositPolicy depositPolicy,
            Clock clock) {
        this.bills = bills;
        this.payments = payments;
        this.refunds = refunds;
        this.appointments = appointments;
        this.providerCustomers = providerCustomers;
        this.processed = processed;
        this.outbox = outbox;
        this.provider = provider;
        this.scheduler = scheduler;
        this.access = access;
        this.tx = tx;
        this.depositPolicy = depositPolicy;
        this.clock = clock;
    }

    /** A payment the browser can now confirm with the Payment Element. */
    public record Started(Payment payment, String clientSecret) {
    }

    // --- Deposit at booking (#86) ------------------------------------------

    /**
     * Starts (or, for a retried Idempotency-Key, resumes) the deposit for one
     * of the caller's bookings. The amount comes from the deposit policy and
     * the booking's estimate, never from the client.
     */
    public Started requestDeposit(long appointmentId, Caller caller, String idempotencyKey) {
        String key = "deposit:" + appointmentId + ":" + requireKey(idempotencyKey);
        Optional<Payment> retried = payments.findByIdempotencyKey(key);
        if (retried.isPresent()) {
            Bill bill = bills.findById(retried.get().billId()).orElseThrow();
            access.requireOwner(caller, bill.customerId(), "appointment");
            return resume(retried.get(), bill);
        }

        AppointmentRef booking = appointments.findById(appointmentId)
                .or(() -> fetchBooking(caller, appointmentId))
                .orElseThrow(() -> new NotFoundException("appointment not found"));
        access.requireOwner(caller, booking.customerId(), "appointment");
        if (booking.status() != AppointmentRef.Status.BOOKED) {
            throw new ConflictException("appointment " + appointmentId + " is " + booking.status()
                    + "; a deposit is only taken for an upcoming booking");
        }
        Money amount = depositPolicy.amountFor(booking.estimate());
        if (!amount.isPositive()) {
            throw new ConflictException("no deposit is due for appointment " + appointmentId);
        }

        Bill bill = openBillFor(booking);
        Object decision = tx.execute(status -> {
            Bill locked = bills.findByIdForUpdate(bill.id()).orElseThrow();
            if (locked.status() != BillStatus.OPEN) {
                throw new ConflictException("appointment " + appointmentId + " is already billed; pay the bill instead");
            }
            List<Payment> deposits = payments.findByBill(locked.id()).stream()
                    .filter(p -> p.type() == PaymentType.DEPOSIT)
                    .toList();
            if (deposits.stream().anyMatch(p -> p.status().isCaptured())) {
                throw new ConflictException("the deposit for appointment " + appointmentId + " is already paid");
            }
            if (deposits.stream().anyMatch(p -> p.status() == PaymentStatus.PROCESSING)) {
                throw new ConflictException("the deposit for appointment " + appointmentId + " is being processed");
            }
            // A deposit started earlier (another tab, a lost response) for the
            // same amount is resumed instead of opening a second intent.
            Optional<Payment> open = deposits.stream()
                    .filter(p -> p.status() == PaymentStatus.REQUIRES_PAYMENT && p.amount().equals(amount))
                    .findFirst();
            if (open.isPresent()) {
                return open.get();
            }
            return payments.save(Payment.requested(locked.id(), PaymentType.DEPOSIT, amount, key));
        });
        return resume((Payment) decision, bill);
    }

    private Optional<AppointmentRef> fetchBooking(Caller caller, long appointmentId) {
        // The AppointmentBooked event can trail the booking by a relay
        // interval; a customer can ask scheduler for their own booking directly.
        if (!caller.isCustomer()) {
            return Optional.empty();
        }
        Optional<AppointmentRef> found = scheduler.findMyAppointment(caller.token(), appointmentId);
        found.ifPresent(appointments::upsert);
        return found;
    }

    private Bill openBillFor(AppointmentRef booking) {
        Optional<Bill> live = bills.findLiveByAppointment(booking.appointmentId());
        if (live.isPresent()) {
            return live.get();
        }
        try {
            return bills.save(Bill.open(booking.appointmentId(), booking.customerId(), booking.vehicleId(),
                    booking.currency()));
        } catch (ConflictException e) {
            // A concurrent request created it first.
            return bills.findLiveByAppointment(booking.appointmentId()).orElseThrow(() -> e);
        }
    }

    // --- Paying the balance (#88) ------------------------------------------

    /**
     * Starts (or resumes) a FINAL payment for the bill's current balance,
     * computed here, never sent by the client. Open intents for another amount
     * (an unpaid deposit, or a balance that has changed) are canceled first so
     * the customer can't pay twice.
     */
    public Started payBalance(long billId, Caller caller, String idempotencyKey) {
        String key = "bill:" + billId + ":" + requireKey(idempotencyKey);
        Optional<Payment> retried = payments.findByIdempotencyKey(key);
        if (retried.isPresent()) {
            Bill bill = bills.findById(retried.get().billId()).orElseThrow();
            access.requireOwner(caller, bill.customerId(), "bill");
            return resume(retried.get(), bill);
        }

        Bill bill = bills.findById(billId).orElseThrow(() -> new NotFoundException("bill not found"));
        access.requireOwner(caller, bill.customerId(), "bill");

        // 1. Under the lock: what's due, and which open intents are stale.
        List<Payment> stale = tx.execute(status -> {
            Bill locked = bills.findByIdForUpdate(billId).orElseThrow();
            Money due = requirePayable(locked);
            return payments.findByBill(billId).stream()
                    .filter(p -> p.status() == PaymentStatus.REQUIRES_PAYMENT)
                    .filter(p -> p.type() == PaymentType.DEPOSIT || !p.amount().equals(due))
                    .toList();
        });

        // 2. Cancel them at the provider (no lock held), then record what it says.
        for (Payment p : stale) {
            cancelAtProvider(p);
        }

        // 3. Under the lock again: the balance may have moved while we were away.
        Payment payment = tx.execute(status -> {
            Bill locked = bills.findByIdForUpdate(billId).orElseThrow();
            Money due = requirePayable(locked);
            Optional<Payment> open = payments.findByBill(billId).stream()
                    .filter(p -> p.type() == PaymentType.FINAL && p.status() == PaymentStatus.REQUIRES_PAYMENT
                            && p.amount().equals(due))
                    .findFirst();
            return open.orElseGet(() -> payments.save(Payment.requested(billId, PaymentType.FINAL, due, key)));
        });
        return resume(payment, bill);
    }

    /** The balance due on a bill that can take a payment now; 409 otherwise. */
    private Money requirePayable(Bill bill) {
        if (bill.status() != BillStatus.PENDING && bill.status() != BillStatus.PARTIALLY_PAID) {
            throw new ConflictException("bill " + bill.id() + " is " + bill.status() + " and takes no payment");
        }
        List<Payment> all = payments.findByBill(bill.id());
        if (all.stream().anyMatch(p -> p.status() == PaymentStatus.PROCESSING)) {
            throw new ConflictException("a payment on bill " + bill.id() + " is still being processed");
        }
        Money due = Balance.of(bill, all, refunds.findByBill(bill.id())).due();
        if (!due.isPositive()) {
            throw new ConflictException("nothing is due on bill " + bill.id());
        }
        return due;
    }

    private void cancelAtProvider(Payment payment) {
        if (payment.stripePaymentIntentId() == null) {
            tx.executeWithoutResult(s -> markFailed(payment.id(), "superseded", "replaced by a newer payment"));
            return;
        }
        ProviderIntent intent;
        try {
            intent = provider.cancelIntent(payment.stripePaymentIntentId());
        } catch (PaymentProviderException e) {
            if (e.kind() != PaymentProviderException.Kind.CONFLICT) {
                throw e;
            }
            // Already succeeded or processing: record that instead, and let
            // step 3 see the new balance.
            intent = provider.retrieveIntent(payment.stripePaymentIntentId());
        }
        ProviderIntent seen = intent;
        tx.executeWithoutResult(s -> apply(payment.id(), seen, false));
    }

    // --- Creating the provider's intent ------------------------------------

    /**
     * Hands back the payment's client secret, creating its intent first if it
     * has none yet. The intent's idempotency key is derived from our payment
     * id, so a retry here can never open a second intent at the provider.
     */
    private Started resume(Payment payment, Bill bill) {
        if (payment.isFinal() || payment.status() == PaymentStatus.FAILED) {
            return new Started(payment, null);
        }
        if (payment.stripePaymentIntentId() != null) {
            return new Started(payment, provider.retrieveIntent(payment.stripePaymentIntentId()).clientSecret());
        }

        ProviderIntent intent;
        try {
            intent = provider.createIntent(payment.amount(), providerCustomer(bill.customerId()), Map.of(
                    "payment_id", Long.toString(payment.id()),
                    "bill_id", Long.toString(bill.id()),
                    "appointment_id", Long.toString(bill.appointmentId()),
                    "type", payment.type().name()), "payment-" + payment.id());
        } catch (PaymentProviderException e) {
            // Nothing exists to pay against; the customer starts over with a new key.
            tx.executeWithoutResult(s -> markFailed(payment.id(), "provider_error", e.getMessage()));
            throw e;
        }

        Payment linked = tx.execute(s -> {
            Payment current = payments.findById(payment.id()).orElseThrow();
            return current.stripePaymentIntentId() == null
                    ? payments.save(current.withIntent(intent.id()))
                    : current;
        });
        return new Started(linked, intent.clientSecret());
    }

    private String providerCustomer(long customerId) {
        Optional<String> known = providerCustomers.findStripeCustomerId(customerId);
        if (known.isPresent()) {
            return known.get();
        }
        String created = provider.createCustomer(customerId, "customer-" + customerId);
        try {
            providerCustomers.save(customerId, created);
            return created;
        } catch (ConflictException e) {
            // A concurrent request saved one first; with the same idempotency
            // key it's the same Stripe customer anyway.
            return providerCustomers.findStripeCustomerId(customerId).orElseThrow(() -> e);
        }
    }

    private void markFailed(long paymentId, String code, String message) {
        Payment p = payments.findById(paymentId).orElseThrow();
        Bill bill = bills.findByIdForUpdate(p.billId()).orElseThrow();
        Payment current = payments.findById(paymentId).orElseThrow();
        if (!current.isFinal() && current.stripePaymentIntentId() == null) {
            payments.save(current.failed(code, message));
        }
        log.info("payment {} on bill {} failed before reaching the provider: {}", paymentId, bill.id(), message);
    }

    // --- What the provider says happened (#83) -----------------------------

    /**
     * A webhook. Its payload isn't trusted for state: the intent is re-read
     * from the provider, so events arriving out of order (a payment_failed
     * after succeeded) can't move a payment backwards. The event id is
     * recorded in the same transaction as the change.
     */
    public void handleWebhook(ProviderEvent event) {
        if (event.intentId() == null || !event.type().startsWith("payment_intent.")) {
            // charge.refunded / charge.dispute.created belong to the refund work (#89).
            tx.executeWithoutResult(s -> processed.markProcessed(STRIPE_SOURCE, event.id()));
            log.info("acknowledged Stripe event {} ({}) without action", event.id(), event.type());
            return;
        }
        ProviderIntent intent = provider.retrieveIntent(event.intentId());
        tx.executeWithoutResult(s -> {
            if (!processed.markProcessed(STRIPE_SOURCE, event.id())) {
                log.debug("Stripe event {} already processed", event.id());
                return;
            }
            Optional<Payment> payment = payments.findByStripePaymentIntentId(intent.id())
                    .or(() -> Optional.ofNullable(intent.paymentId()).flatMap(payments::findById));
            if (payment.isEmpty()) {
                log.warn("Stripe event {} ({}) is for intent {}, which isn't one of ours", event.id(), event.type(),
                        intent.id());
                return;
            }
            apply(payment.get().id(), intent, true);
        });
    }

    /** Re-reads a payment's intent and applies it: recovers a missed webhook. Staff only. */
    public Payment sync(long paymentId, Caller caller) {
        if (!caller.isStaff()) {
            throw new NotFoundException("payment not found");
        }
        Payment payment = payments.findById(paymentId).orElseThrow(() -> new NotFoundException("payment not found"));
        if (payment.stripePaymentIntentId() == null) {
            return payment;
        }
        ProviderIntent intent = provider.retrieveIntent(payment.stripePaymentIntentId());
        return tx.execute(s -> apply(paymentId, intent, true));
    }

    /**
     * Test mode only: pays with a provider test method (pm_card_visa,
     * pm_card_chargeDeclined, ...) and applies the result as the webhook would.
     * Lets the whole flow run without a browser.
     */
    public Payment confirmTest(long paymentId, String paymentMethod, Caller caller) {
        Payment payment = getPayment(paymentId, caller);
        if (payment.stripePaymentIntentId() == null || !payment.isOpen()) {
            throw new ConflictException("payment " + paymentId + " is " + payment.status() + " and can't be confirmed");
        }
        ProviderIntent intent = provider.confirmTestIntent(payment.stripePaymentIntentId(),
                paymentMethod == null || paymentMethod.isBlank() ? "pm_card_visa" : paymentMethod);
        return tx.execute(s -> apply(paymentId, intent, true));
    }

    public Payment getPayment(long paymentId, Caller caller) {
        Payment payment = payments.findById(paymentId).orElseThrow(() -> new NotFoundException("payment not found"));
        Bill bill = bills.findById(payment.billId()).orElseThrow();
        access.requireCanRead(caller, bill.customerId(), "payment");
        return payment;
    }

    /**
     * Moves the payment to the intent's state and settles the bill. Must run
     * inside a transaction; takes the bill's row lock.
     *
     * @param notify whether a decline is published (not for intents we canceled ourselves)
     */
    Payment apply(long paymentId, ProviderIntent intent, boolean notify) {
        Payment found = payments.findById(paymentId).orElseThrow();
        Bill bill = bills.findByIdForUpdate(found.billId()).orElseThrow();
        Payment payment = payments.findById(paymentId).orElseThrow(); // re-read under the lock
        if (payment.isFinal()) {
            // Only one final state can still move: a payment we canceled whose
            // intent the provider charged anyway. The money is real, so the bill
            // must reflect it (a refund may then be due). Refunded payments
            // keep a succeeded intent and must not be reset.
            boolean chargedAfterCancel = payment.status() == PaymentStatus.CANCELED && "succeeded".equals(intent.status());
            if (!chargedAfterCancel) {
                return payment;
            }
            log.error("payment {} was canceled but its intent {} succeeded; recording it", paymentId, intent.id());
        }
        if (payment.stripePaymentIntentId() == null) {
            payment = payment.withIntent(intent.id());
        }

        Payment next = switch (intent.status()) {
            case "succeeded" -> {
                if (intent.amountReceived() != payment.amount().amountMinor()) {
                    log.error("payment {} expected {} but intent {} received {} minor units", paymentId,
                            payment.amount(), intent.id(), intent.amountReceived());
                }
                yield payment.succeeded(Instant.now(clock), intent.cardBrand(), intent.cardLast4());
            }
            case "processing" -> payment.processing();
            case "canceled" -> payment.canceled(intent.failureCode(), intent.failureMessage());
            case "requires_payment_method" -> intent.failureCode() == null
                    ? payment
                    : payment.declined(intent.failureCode(), intent.failureMessage());
            // requires_action / requires_confirmation / requires_capture: waiting on the customer.
            default -> payment;
        };
        if (next.equals(payment) && Objects.equals(payment.stripePaymentIntentId(), found.stripePaymentIntentId())) {
            return payment;
        }
        Payment saved = payments.save(next);

        if (saved.status() == PaymentStatus.SUCCEEDED && payment.status() != PaymentStatus.SUCCEEDED) {
            settle(bill, saved);
        } else if (notify && saved.status() == PaymentStatus.REQUIRES_PAYMENT && saved.failureCode() != null
                && !(Objects.equals(saved.failureCode(), found.failureCode())
                        && Objects.equals(saved.failureMessage(), found.failureMessage()))) {
            outbox.append(BillingEvents.paymentFailed(bill, saved));
        }
        return saved;
    }

    /** Recomputes the bill after a successful payment and publishes PaymentReceived. */
    private void settle(Bill bill, Payment succeeded) {
        List<Payment> all = payments.findByBill(bill.id());
        Balance balance = Balance.of(bill, all, refunds.findByBill(bill.id()));
        Bill updated = bill;
        BillStatus status = balance.statusFor(bill);
        if (status != bill.status()) {
            updated = bills.save(bill.withStatus(status));
        }
        // An OPEN bill isn't priced, so a deposit on it is credit, not an overpayment.
        if (updated.status() != BillStatus.OPEN && balance.refundDue().isPositive()) {
            log.warn("bill {} is overpaid by {}; a refund is due (#89)", bill.id(), balance.refundDue());
        }
        outbox.append(BillingEvents.paymentReceived(updated, succeeded, balance));
    }

    private static String requireKey(String idempotencyKey) {
        if (idempotencyKey == null || idempotencyKey.isBlank()) {
            throw new IllegalArgumentException("the Idempotency-Key header is required");
        }
        if (idempotencyKey.length() > 200) {
            throw new IllegalArgumentException("Idempotency-Key must be at most 200 characters");
        }
        return idempotencyKey.trim();
    }
}
