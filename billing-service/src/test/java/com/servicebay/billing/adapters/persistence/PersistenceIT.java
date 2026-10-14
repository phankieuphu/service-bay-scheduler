package com.servicebay.billing.adapters.persistence;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.servicebay.billing.TestcontainersConfiguration;
import com.servicebay.billing.domain.balance.Balance;
import com.servicebay.billing.domain.bill.Bill;
import com.servicebay.billing.domain.bill.BillLine;
import com.servicebay.billing.domain.bill.BillStatus;
import com.servicebay.billing.domain.bill.LineKind;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.outbox.OutboxMessage;
import com.servicebay.billing.domain.payment.Payment;
import com.servicebay.billing.domain.payment.PaymentStatus;
import com.servicebay.billing.domain.payment.PaymentType;
import com.servicebay.billing.domain.payment.Refund;
import com.servicebay.billing.domain.payment.RefundStatus;
import com.servicebay.billing.domain.ports.BillRepository;
import com.servicebay.billing.domain.ports.ConflictException;
import com.servicebay.billing.domain.ports.OutboxRepository;
import com.servicebay.billing.domain.ports.PaymentRepository;
import com.servicebay.billing.domain.ports.ProcessedEventRepository;
import com.servicebay.billing.domain.ports.RefundRepository;
import com.servicebay.billing.domain.ports.StripeCustomerRepository;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.List;
import java.util.concurrent.atomic.AtomicLong;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.context.annotation.Import;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.transaction.IllegalTransactionStateException;
import org.springframework.transaction.support.TransactionTemplate;

/**
 * The Flyway schema plus the JPA adapters against a real Postgres. The
 * context only starting means Flyway applied V1 and Hibernate's validate
 * found every mapping matches it.
 */
@SpringBootTest
@Import(TestcontainersConfiguration.class)
class PersistenceIT {

    private static final AtomicLong APPOINTMENTS = new AtomicLong(System.nanoTime() % 1_000_000_000);

    @Autowired BillRepository bills;
    @Autowired PaymentRepository payments;
    @Autowired RefundRepository refunds;
    @Autowired StripeCustomerRepository stripeCustomers;
    @Autowired ProcessedEventRepository processedEvents;
    @Autowired OutboxRepository outbox;
    @Autowired TransactionTemplate tx;
    @Autowired JdbcClient jdbc;

    private static long nextAppointment() {
        return APPOINTMENTS.incrementAndGet();
    }

    @Test
    void billRoundTripsWithItsLines() {
        long appointment = nextAppointment();
        Bill open = bills.save(Bill.open(appointment, 20, 30, "VND"));
        assertThat(open.id()).isNotNull();
        assertThat(open.status()).isEqualTo(BillStatus.OPEN);

        Instant at = Instant.now().truncatedTo(ChronoUnit.MICROS);
        Bill priced = bills.save(open.price(List.of(
                new BillLine(null, LineKind.SERVICE, 5L, "Oil change", 1, Money.of(800_000, "VND")),
                new BillLine(null, LineKind.MATERIAL, 9L, "Filter", 2, Money.of(100_000, "VND"))), at));

        Bill loaded = bills.findById(priced.id()).orElseThrow();
        assertThat(loaded.status()).isEqualTo(BillStatus.PENDING);
        assertThat(loaded.total()).isEqualTo(Money.of(1_000_000, "VND"));
        assertThat(loaded.finalizedAt()).isEqualTo(at);
        assertThat(loaded.lines()).extracting(BillLine::name).containsExactly("Oil change", "Filter");
        assertThat(loaded.lines()).allMatch(l -> l.id() != null);
        assertThat(loaded.version()).isEqualTo(1);
        assertThat(bills.findLiveByAppointment(appointment)).map(Bill::id).contains(loaded.id());

        // the generated total_minor column agrees with the domain's sum
        Long dbTotal = jdbc.sql("select total_minor from bill where id = :id").param("id", loaded.id()).query(Long.class).single();
        assertThat(dbTotal).isEqualTo(1_000_000L);
    }

    @Test
    void oneLiveBillPerAppointmentButAVoidedOneCanBeReplaced() {
        long appointment = nextAppointment();
        Bill first = bills.save(Bill.open(appointment, 20, 30, "USD"));

        assertThatThrownBy(() -> bills.save(Bill.open(appointment, 20, 30, "USD")))
                .isInstanceOf(ConflictException.class);

        bills.save(first.withStatus(BillStatus.VOID));
        Bill second = bills.save(Bill.open(appointment, 20, 30, "USD"));
        assertThat(second.id()).isNotEqualTo(first.id());
        assertThat(bills.findLiveByAppointment(appointment)).map(Bill::id).contains(second.id());
    }

    @Test
    void staleVersionIsAConflict() {
        Bill bill = bills.save(Bill.open(nextAppointment(), 20, 30, "USD"));
        bills.save(bill.withStatus(BillStatus.VOID));

        assertThatThrownBy(() -> bills.save(bill.price(List.of(), Instant.now())))
                .isInstanceOf(ConflictException.class);
    }

    @Test
    void paymentsAndRefundsFeedTheBalance() {
        Bill bill = bills.save(Bill.open(nextAppointment(), 20, 30, "USD"));
        Payment deposit = payments.save(Payment.requested(bill.id(), PaymentType.DEPOSIT, Money.of(5_000, "USD"), "dep-" + bill.id()));
        assertThat(deposit.status()).isEqualTo(PaymentStatus.REQUIRES_PAYMENT);

        Payment succeeded = payments.save(new Payment(deposit.id(), bill.id(), deposit.type(), deposit.amount(),
                PaymentStatus.SUCCEEDED, "pi_" + bill.id(), deposit.idempotencyKey(), null, null, "visa", "4242",
                Instant.now(), deposit.version()));
        refunds.save(new Refund(null, succeeded.id(), Money.of(1_000, "USD"), "re_" + bill.id(), RefundStatus.SUCCEEDED, "goodwill", 0));

        assertThat(payments.findByStripePaymentIntentId("pi_" + bill.id())).isPresent();
        assertThat(payments.findByIdempotencyKey("dep-" + bill.id())).isPresent();
        Balance balance = Balance.of(bill, payments.findByBill(bill.id()), refunds.findByBill(bill.id()));
        assertThat(balance.held()).isEqualTo(Money.of(4_000, "USD"));
    }

    @Test
    void duplicateIdempotencyKeyIsAConflict() {
        Bill bill = bills.save(Bill.open(nextAppointment(), 20, 30, "USD"));
        payments.save(Payment.requested(bill.id(), PaymentType.DEPOSIT, Money.of(100, "USD"), "dup-" + bill.id()));

        assertThatThrownBy(() -> payments.save(Payment.requested(bill.id(), PaymentType.DEPOSIT, Money.of(100, "USD"), "dup-" + bill.id())))
                .isInstanceOf(ConflictException.class);
    }

    @Test
    void databaseRejectsNonPositiveAmounts() {
        Bill bill = bills.save(Bill.open(nextAppointment(), 20, 30, "USD"));
        assertThatThrownBy(() -> jdbc.sql("""
                insert into payment (bill_id, type, amount_minor, currency, status, idempotency_key)
                values (:bill, 'DEPOSIT', 0, 'USD', 'REQUIRES_PAYMENT', :key)""")
                .param("bill", bill.id()).param("key", "zero-" + bill.id()).update())
                .isInstanceOf(DataIntegrityViolationException.class);
    }

    @Test
    void processedEventDeduplicates() {
        String id = "evt_" + nextAppointment();
        Boolean first = tx.execute(s -> processedEvents.markProcessed("stripe", id));
        Boolean again = tx.execute(s -> processedEvents.markProcessed("stripe", id));
        Boolean otherSource = tx.execute(s -> processedEvents.markProcessed("scheduler.appointment.service-completed.v1", id));
        assertThat(first).isTrue();
        assertThat(again).isFalse();
        assertThat(otherSource).isTrue();
    }

    @Test
    void outboxRefusesToWriteOutsideATransaction() {
        assertThatThrownBy(() -> outbox.append(OutboxMessage.of("t", "k", "{}")))
                .isInstanceOf(IllegalTransactionStateException.class);

        tx.executeWithoutResult(s -> outbox.append(OutboxMessage.of("billing.test.v1", "42", "{\"a\":1}")));
        Integer count = jdbc.sql("select count(*) from outbox_message where topic = 'billing.test.v1' and payload->>'a' = '1'")
                .query(Integer.class).single();
        assertThat(count).isPositive();
    }

    @Test
    void stripeCustomerIsCreatedOnce() {
        long customer = nextAppointment();
        assertThat(stripeCustomers.findStripeCustomerId(customer)).isEmpty();
        stripeCustomers.save(customer, "cus_" + customer);
        assertThat(stripeCustomers.findStripeCustomerId(customer)).contains("cus_" + customer);
        assertThatThrownBy(() -> stripeCustomers.save(customer, "cus_other")).isInstanceOf(ConflictException.class);
    }

    @Test
    void findByIdForUpdateNeedsATransactionAndReturnsTheBill() {
        Bill bill = bills.save(Bill.open(nextAppointment(), 20, 30, "USD"));
        Bill locked = tx.execute(s -> bills.findByIdForUpdate(bill.id()).orElseThrow());
        assertThat(locked.id()).isEqualTo(bill.id());
    }
}
