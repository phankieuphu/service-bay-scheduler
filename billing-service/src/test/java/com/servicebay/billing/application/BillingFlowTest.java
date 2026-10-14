package com.servicebay.billing.application;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.servicebay.billing.application.Bookings.CompletedService;
import com.servicebay.billing.application.Bookings.Completion;
import com.servicebay.billing.domain.bill.Bill;
import com.servicebay.billing.domain.bill.BillStatus;
import com.servicebay.billing.domain.bill.LineKind;
import com.servicebay.billing.domain.booking.AppointmentRef;
import com.servicebay.billing.domain.deposit.DepositPolicy;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.payment.Payment;
import com.servicebay.billing.domain.payment.PaymentStatus;
import com.servicebay.billing.domain.payment.PaymentType;
import com.servicebay.billing.domain.ports.ConflictException;
import com.servicebay.billing.domain.ports.Directories.UpstreamException;
import com.servicebay.billing.domain.ports.Directories.VehicleDirectory.Warranty;
import com.servicebay.billing.domain.ports.PaymentProvider.ProviderEvent;
import com.servicebay.billing.domain.ports.PaymentProviderException;
import java.math.BigDecimal;
import java.time.Clock;
import java.time.Instant;
import java.time.LocalDate;
import java.time.ZoneOffset;
import java.util.List;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.transaction.support.TransactionOperations;

/** Deposit → completion → balance, driven through the use cases with in-memory ports. */
class BillingFlowTest {

    private static final String BOOKED = "scheduler.appointment.booked.v1";
    private static final String CANCELLED = "scheduler.appointment.cancelled.v1";
    private static final String COMPLETED = "scheduler.appointment.service-completed.v1";

    private final Caller jane = new Caller("u-jane", "CUSTOMER", "tok-jane");
    private final Caller mallory = new Caller("u-mallory", "CUSTOMER", "tok-mallory");
    private final Caller admin = new Caller("u-admin", "ADMIN", "tok-admin");

    // 105.50 USD of booked services: Oil change 45.00 + Brake inspection 60.50.
    private final AppointmentRef booking = new AppointmentRef(500, 42, 7, 1, Money.of(10_550, "USD"),
            AppointmentRef.Status.BOOKED);

    private Fakes.Bills bills;
    private Fakes.Payments payments;
    private Fakes.Appointments appointments;
    private Fakes.Outbox outbox;
    private Fakes.Provider provider;
    private Fakes.Vehicles vehicles;
    private Fakes.Scheduler scheduler;
    private Payments paymentUseCases;
    private Bookings bookings;
    private Bills billQueries;

    @BeforeEach
    void setUp() {
        bills = new Fakes.Bills();
        payments = new Fakes.Payments();
        appointments = new Fakes.Appointments();
        outbox = new Fakes.Outbox();
        provider = new Fakes.Provider();
        vehicles = new Fakes.Vehicles();
        scheduler = new Fakes.Scheduler();
        Fakes.Refunds refunds = new Fakes.Refunds();
        Fakes.Processed processed = new Fakes.Processed();
        Fakes.Customers customers = new Fakes.Customers();
        customers.byToken.put("tok-jane", 42L);
        customers.byToken.put("tok-mallory", 43L);
        Access access = new Access(customers);
        TransactionOperations tx = TransactionOperations.withoutTransaction();
        Clock clock = Clock.fixed(Instant.parse("2026-10-20T04:00:00Z"), ZoneOffset.UTC);
        BillingProperties props = new BillingProperties("USD", new BigDecimal("50.00"), "Asia/Ho_Chi_Minh",
                "PERCENT", BigDecimal.valueOf(20));

        paymentUseCases = new Payments(bills, payments, refunds, appointments, new Fakes.ProviderCustomers(),
                processed, outbox, provider, scheduler, access, tx,
                DepositPolicy.parse(props.depositPolicy(), props.depositValue()), clock);
        bookings = new Bookings(appointments, bills, payments, refunds, processed, outbox, vehicles, provider,
                paymentUseCases, tx, props, clock);
        billQueries = new Bills(bills, payments, refunds, access);
    }

    private Completion completion(String completedAt) {
        return new Completion(500, 42, 7, 1, Instant.parse(completedAt), "USD", List.of(
                new CompletedService(1, "Oil change", 1, 4_500),
                new CompletedService(3, "Brake inspection", 1, 6_050)));
    }

    private Bill billOf(long appointmentId) {
        return bills.findLiveByAppointment(appointmentId).orElseThrow();
    }

    private void webhook(String eventId, String intentId) {
        paymentUseCases.handleWebhook(new ProviderEvent(eventId, "payment_intent.succeeded", intentId));
    }

    // --- Deposit -------------------------------------------------------------

    @Test
    void depositIsThePolicysShareOfTheEstimateAndIdempotent() {
        bookings.booked(BOOKED, "e1", booking);

        Payments.Started first = paymentUseCases.requestDeposit(500, jane, "key-1");

        assertThat(first.payment().type()).isEqualTo(PaymentType.DEPOSIT);
        assertThat(first.payment().amount()).isEqualTo(Money.of(2_110, "USD")); // 20% of 105.50
        assertThat(first.clientSecret()).isEqualTo("pi_1_secret");
        assertThat(billOf(500).status()).isEqualTo(BillStatus.OPEN);

        // Same key: same payment. A new key while it's still open: resumed, not a second intent.
        assertThat(paymentUseCases.requestDeposit(500, jane, "key-1").payment().id()).isEqualTo(first.payment().id());
        assertThat(paymentUseCases.requestDeposit(500, jane, "key-2").payment().id()).isEqualTo(first.payment().id());
        assertThat(provider.creates).isEqualTo(1);
    }

    @Test
    void depositBeforeTheBookedEventAsksScheduler() {
        scheduler.mine.put(500L, booking);

        Payments.Started started = paymentUseCases.requestDeposit(500, jane, "key-1");

        assertThat(started.payment().amount().amountMinor()).isEqualTo(2_110);
        assertThat(appointments.findById(500)).isPresent();
    }

    @Test
    void depositRejections() {
        bookings.booked(BOOKED, "e1", booking);

        assertThatThrownBy(() -> paymentUseCases.requestDeposit(500, mallory, "k")).isInstanceOf(NotFoundException.class);
        assertThatThrownBy(() -> paymentUseCases.requestDeposit(999, jane, "k")).isInstanceOf(NotFoundException.class);
        assertThatThrownBy(() -> paymentUseCases.requestDeposit(500, jane, " ")).isInstanceOf(IllegalArgumentException.class);

        Payment deposit = paymentUseCases.requestDeposit(500, jane, "k").payment();
        provider.succeed(deposit.stripePaymentIntentId());
        webhook("evt_1", deposit.stripePaymentIntentId());
        assertThatThrownBy(() -> paymentUseCases.requestDeposit(500, jane, "k2"))
                .isInstanceOf(ConflictException.class).hasMessageContaining("already paid");
    }

    @Test
    void paidDepositPublishesPaymentReceivedWithNoBalanceYet() {
        bookings.booked(BOOKED, "e1", booking);
        Payment deposit = paymentUseCases.requestDeposit(500, jane, "k").payment();

        provider.succeed(deposit.stripePaymentIntentId());
        webhook("evt_1", deposit.stripePaymentIntentId());
        webhook("evt_1", deposit.stripePaymentIntentId()); // Stripe redelivers

        Payment paid = payments.findById(deposit.id()).orElseThrow();
        assertThat(paid.status()).isEqualTo(PaymentStatus.SUCCEEDED);
        assertThat(paid.cardLast4()).isEqualTo("4242");
        assertThat(billOf(500).status()).isEqualTo(BillStatus.OPEN);
        assertThat(outbox.topics()).containsExactly(BillingEvents.TOPIC_PAYMENT_RECEIVED);
        assertThat(outbox.messages.get(0).payload()).contains("\"type\":\"DEPOSIT\"", "\"balance_due_minor\":null");
        assertThat(outbox.messages.get(0).key()).isEqualTo("500");
    }

    // --- Pricing on ServiceCompleted (#85) -----------------------------------

    @Test
    void completionPricesTheBillWithTheWarrantyFeeAndCountsTheDeposit() {
        bookings.booked(BOOKED, "e1", booking);
        Payment deposit = paymentUseCases.requestDeposit(500, jane, "k").payment();
        provider.succeed(deposit.stripePaymentIntentId());
        webhook("evt_1", deposit.stripePaymentIntentId());

        // 18:00 UTC on the 19th is already the 20th in Saigon.
        bookings.completed(COMPLETED, "e2", completion("2026-10-19T18:00:00Z"));

        Bill bill = billOf(500);
        assertThat(vehicles.askedFor).isEqualTo(LocalDate.of(2026, 10, 20));
        assertThat(bill.lines()).extracting(l -> l.kind()).containsExactly(LineKind.WARRANTY_FEE, LineKind.SERVICE,
                LineKind.SERVICE);
        assertThat(bill.lines().get(0).name()).contains("no warranty on record");
        assertThat(bill.total()).isEqualTo(Money.of(15_550, "USD")); // 50.00 + 45.00 + 60.50
        assertThat(bill.status()).isEqualTo(BillStatus.PARTIALLY_PAID);
        assertThat(outbox.topics()).containsExactly(BillingEvents.TOPIC_PAYMENT_RECEIVED, BillingEvents.TOPIC_BILL_READY);
        assertThat(outbox.messages.get(1).payload()).contains("\"balance_due_minor\":13440");
        assertThat(billQueries.forAppointment(500, jane).balance().due()).isEqualTo(Money.of(13_440, "USD"));
    }

    @Test
    void inWarrantyMeansNoFee() {
        vehicles.answer = new Warranty(true, LocalDate.of(2029, 1, 12));

        bookings.completed(COMPLETED, "e2", completion("2026-10-20T04:00:00Z"));

        assertThat(billOf(500).lines()).extracting(l -> l.kind()).doesNotContain(LineKind.WARRANTY_FEE);
        assertThat(billOf(500).total()).isEqualTo(Money.of(10_550, "USD"));
        assertThat(billOf(500).status()).isEqualTo(BillStatus.PENDING);
    }

    @Test
    void completionIsIdempotent() {
        bookings.completed(COMPLETED, "e2", completion("2026-10-20T04:00:00Z"));
        bookings.completed(COMPLETED, "e2", completion("2026-10-20T04:00:00Z"));
        bookings.completed(COMPLETED, "e3", completion("2026-10-20T04:00:00Z")); // a different event, same appointment

        assertThat(bills.rows).hasSize(1);
        assertThat(billOf(500).lines()).hasSize(3);
        assertThat(outbox.topics()).containsExactly(BillingEvents.TOPIC_BILL_READY);
    }

    @Test
    void anUnansweredWarrantyLookupIsRetriedNotAssumed() {
        vehicles.down = true;
        assertThatThrownBy(() -> bookings.completed(COMPLETED, "e2", completion("2026-10-20T04:00:00Z")))
                .isInstanceOf(UpstreamException.class);
        assertThat(bills.rows).isEmpty();

        vehicles.down = false; // the consumer's retry
        bookings.completed(COMPLETED, "e2", completion("2026-10-20T04:00:00Z"));
        assertThat(billOf(500).status()).isEqualTo(BillStatus.PENDING);
    }

    @Test
    void aDepositCoveringTheBillLeavesItPaid() {
        bookings.booked(BOOKED, "e1", new AppointmentRef(500, 42, 7, 1, Money.of(10_550, "USD"),
                AppointmentRef.Status.BOOKED));
        vehicles.answer = new Warranty(true, null);
        Completion cheap = new Completion(500, 42, 7, 1, Instant.parse("2026-10-20T04:00:00Z"), "USD",
                List.of(new CompletedService(4, "Battery test", 1, 2_000)));
        Payment deposit = paymentUseCases.requestDeposit(500, jane, "k").payment(); // 21.10
        provider.succeed(deposit.stripePaymentIntentId());
        webhook("evt_1", deposit.stripePaymentIntentId());

        bookings.completed(COMPLETED, "e2", cheap);

        assertThat(billOf(500).status()).isEqualTo(BillStatus.PAID);
        assertThat(outbox.topics()).doesNotContain(BillingEvents.TOPIC_BILL_READY);
        assertThat(billQueries.forAppointment(500, jane).balance().refundDue()).isEqualTo(Money.of(110, "USD"));
    }

    // --- Paying the balance (#88) --------------------------------------------

    @Test
    void payingTheBalanceCancelsAnUnpaidDepositAndChargesWhatIsDue() {
        bookings.booked(BOOKED, "e1", booking);
        Payment deposit = paymentUseCases.requestDeposit(500, jane, "dep").payment(); // never paid
        bookings.completed(COMPLETED, "e2", completion("2026-10-20T04:00:00Z"));
        Bill bill = billOf(500);

        Payments.Started started = paymentUseCases.payBalance(bill.id(), jane, "pay-1");

        assertThat(provider.canceled).containsExactly(deposit.stripePaymentIntentId());
        assertThat(payments.findById(deposit.id()).orElseThrow().status()).isEqualTo(PaymentStatus.CANCELED);
        assertThat(started.payment().type()).isEqualTo(PaymentType.FINAL);
        assertThat(started.payment().amount()).isEqualTo(Money.of(15_550, "USD"));

        // A declined card: the payment stays open with the reason, and the UI is told.
        Payment declined = paymentUseCases.confirmTest(started.payment().id(), "pm_card_chargeDeclined", jane);
        assertThat(declined.status()).isEqualTo(PaymentStatus.REQUIRES_PAYMENT);
        assertThat(declined.failureMessage()).isEqualTo("Your card was declined.");
        assertThat(outbox.topics()).endsWith(BillingEvents.TOPIC_PAYMENT_FAILED);

        Payment paid = paymentUseCases.confirmTest(started.payment().id(), "pm_card_visa", jane);
        assertThat(paid.status()).isEqualTo(PaymentStatus.SUCCEEDED);
        assertThat(billOf(500).status()).isEqualTo(BillStatus.PAID);
        assertThat(outbox.topics()).endsWith(BillingEvents.TOPIC_PAYMENT_RECEIVED);
        assertThat(outbox.messages.get(outbox.messages.size() - 1).payload())
                .contains("\"type\":\"FINAL\"", "\"bill_status\":\"PAID\"", "\"balance_due_minor\":0");

        assertThatThrownBy(() -> paymentUseCases.payBalance(bill.id(), jane, "pay-2"))
                .isInstanceOf(ConflictException.class).hasMessageContaining("PAID");
    }

    @Test
    void payBalanceRejections() {
        bookings.booked(BOOKED, "e1", booking);
        paymentUseCases.requestDeposit(500, jane, "dep");
        Bill open = billOf(500);

        assertThatThrownBy(() -> paymentUseCases.payBalance(open.id(), jane, "k"))
                .isInstanceOf(ConflictException.class).hasMessageContaining("OPEN");
        assertThatThrownBy(() -> paymentUseCases.payBalance(open.id(), mallory, "k")).isInstanceOf(NotFoundException.class);
        assertThatThrownBy(() -> paymentUseCases.payBalance(open.id(), admin, "k")).isInstanceOf(NotFoundException.class);
        assertThatThrownBy(() -> paymentUseCases.payBalance(404, jane, "k")).isInstanceOf(NotFoundException.class);
    }

    @Test
    void aProviderOutageFailsThePaymentAndANewKeyStartsOver() {
        bookings.completed(COMPLETED, "e2", completion("2026-10-20T04:00:00Z"));
        long billId = billOf(500).id();
        provider.failCreate = true;

        assertThatThrownBy(() -> paymentUseCases.payBalance(billId, jane, "k1"))
                .isInstanceOf(PaymentProviderException.class);
        assertThat(payments.rows.values()).extracting(Payment::status).containsExactly(PaymentStatus.FAILED);

        provider.failCreate = false;
        Payments.Started retry = paymentUseCases.payBalance(billId, jane, "k2");
        assertThat(retry.clientSecret()).isNotNull();
        assertThat(retry.payment().amount()).isEqualTo(Money.of(15_550, "USD"));
    }

    @Test
    void aChargeOnAnIntentWeCanceledIsStillRecorded() {
        bookings.completed(COMPLETED, "e2", completion("2026-10-20T04:00:00Z"));
        Payment p = paymentUseCases.payBalance(billOf(500).id(), jane, "k").payment();
        provider.cancelIntent(p.stripePaymentIntentId());
        paymentUseCases.sync(p.id(), admin);
        assertThat(payments.findById(p.id()).orElseThrow().status()).isEqualTo(PaymentStatus.CANCELED);

        provider.succeed(p.stripePaymentIntentId()); // the provider charged it anyway
        webhook("evt_late", p.stripePaymentIntentId());

        assertThat(payments.findById(p.id()).orElseThrow().status()).isEqualTo(PaymentStatus.SUCCEEDED);
        assertThat(billOf(500).status()).isEqualTo(BillStatus.PAID);
    }

    // --- Cancellation --------------------------------------------------------

    @Test
    void cancellingABookingVoidsItsUnpaidBill() {
        bookings.booked(BOOKED, "e1", booking);
        Payment deposit = paymentUseCases.requestDeposit(500, jane, "dep").payment();
        Bill bill = billOf(500);

        bookings.cancelled(CANCELLED, "e9", booking.withStatus(AppointmentRef.Status.CANCELLED));

        assertThat(provider.canceled).containsExactly(deposit.stripePaymentIntentId());
        assertThat(bills.findById(bill.id()).orElseThrow().status()).isEqualTo(BillStatus.VOID);
        assertThat(appointments.findById(500).orElseThrow().status()).isEqualTo(AppointmentRef.Status.CANCELLED);
    }

    @Test
    void cancellingAfterAPaidDepositKeepsTheBillOpenForARefundDecision() {
        bookings.booked(BOOKED, "e1", booking);
        Payment deposit = paymentUseCases.requestDeposit(500, jane, "dep").payment();
        provider.succeed(deposit.stripePaymentIntentId());
        webhook("evt_1", deposit.stripePaymentIntentId());

        bookings.cancelled(CANCELLED, "e9", booking.withStatus(AppointmentRef.Status.CANCELLED));

        assertThat(billOf(500).status()).isEqualTo(BillStatus.OPEN);
    }

    // --- Reads (#87, #90) ----------------------------------------------------

    @Test
    void customersSeeOnlyTheirOwnBillsAndStaffSeeAll() {
        bookings.completed(COMPLETED, "e2", completion("2026-10-20T04:00:00Z"));
        long billId = billOf(500).id();

        assertThat(billQueries.get(billId, jane).bill().id()).isEqualTo(billId);
        assertThat(billQueries.get(billId, admin).bill().id()).isEqualTo(billId);
        assertThatThrownBy(() -> billQueries.get(billId, mallory)).isInstanceOf(NotFoundException.class);
        assertThatThrownBy(() -> billQueries.forCustomer(42, null, null, mallory)).isInstanceOf(NotFoundException.class);
        assertThat(billQueries.mine(null, null, jane).bills()).hasSize(1);
        assertThat(billQueries.mine(null, null, mallory).bills()).isEmpty();
    }

    @Test
    void billHistoryPagesNewestFirst() {
        for (long appointment = 1; appointment <= 5; appointment++) {
            bookings.completed(COMPLETED, "c" + appointment, new Completion(appointment, 42, 7, 1,
                    Instant.parse("2026-10-20T04:00:00Z"), "USD", List.of(new CompletedService(1, "Oil change", 1, 4_500))));
        }

        Bills.Page first = billQueries.mine(null, 2, jane);
        Bills.Page second = billQueries.mine(first.nextCursor(), 2, jane);
        Bills.Page last = billQueries.mine(second.nextCursor(), 2, jane);

        assertThat(first.bills()).extracting(v -> v.bill().appointmentId()).containsExactly(5L, 4L);
        assertThat(second.bills()).extracting(v -> v.bill().appointmentId()).containsExactly(3L, 2L);
        assertThat(last.bills()).extracting(v -> v.bill().appointmentId()).containsExactly(1L);
        assertThat(last.nextCursor()).isNull();
    }
}
