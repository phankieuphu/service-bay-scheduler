package com.servicebay.billing.domain.balance;

import static org.assertj.core.api.Assertions.assertThat;

import com.servicebay.billing.domain.bill.Bill;
import com.servicebay.billing.domain.bill.BillLine;
import com.servicebay.billing.domain.bill.BillStatus;
import com.servicebay.billing.domain.bill.LineKind;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.payment.Payment;
import com.servicebay.billing.domain.payment.PaymentStatus;
import com.servicebay.billing.domain.payment.PaymentType;
import com.servicebay.billing.domain.payment.Refund;
import com.servicebay.billing.domain.payment.RefundStatus;
import java.time.Instant;
import java.util.List;
import org.junit.jupiter.api.Test;

class BalanceTest {

    private static final String VND = "VND";

    private static Bill openBill() {
        return withId(Bill.open(10, 20, 30, VND));
    }

    /** A bill priced at 1 000 000 VND: 800 000 service + 200 000 material. */
    private static Bill pricedBill() {
        return openBill().price(List.of(
                new BillLine(null, LineKind.SERVICE, 1L, "Oil change", 1, vnd(800_000)),
                new BillLine(null, LineKind.MATERIAL, 7L, "Filter", 2, vnd(100_000))), Instant.EPOCH);
    }

    private static Bill withId(Bill b) {
        return new Bill(1L, b.appointmentId(), b.customerId(), b.vehicleId(), b.currency(), b.warrantyFee(),
                b.serviceFee(), b.materialCost(), b.status(), b.finalizedAt(), b.lines(), b.version());
    }

    private static Money vnd(long amount) {
        return Money.of(amount, VND);
    }

    private static Payment payment(long id, PaymentType type, long amount, PaymentStatus status) {
        return new Payment(id, 1, type, vnd(amount), status, "pi_" + id, "key-" + id, null, null, 0);
    }

    private static Refund refund(long paymentId, long amount, RefundStatus status) {
        return new Refund(null, paymentId, vnd(amount), "re_" + paymentId, status, null, 0);
    }

    @Test
    void depositOnlyIsCreditOnAnUnpricedBill() {
        Bill bill = openBill();
        Balance b = Balance.of(bill, List.of(payment(1, PaymentType.DEPOSIT, 300_000, PaymentStatus.SUCCEEDED)), List.of());

        assertThat(b.total()).isEqualTo(vnd(0));
        assertThat(b.balance()).isEqualTo(vnd(-300_000));
        assertThat(b.due()).isEqualTo(vnd(0));
        assertThat(b.statusFor(bill)).isEqualTo(BillStatus.OPEN);
    }

    @Test
    void depositCountsTowardThePricedBill() {
        Bill bill = pricedBill();
        Balance b = Balance.of(bill, List.of(payment(1, PaymentType.DEPOSIT, 300_000, PaymentStatus.SUCCEEDED)), List.of());

        assertThat(bill.total()).isEqualTo(vnd(1_000_000));
        assertThat(b.due()).isEqualTo(vnd(700_000));
        assertThat(b.statusFor(bill)).isEqualTo(BillStatus.PARTIALLY_PAID);
    }

    @Test
    void pricedWithNothingPaidIsPending() {
        Bill bill = pricedBill();
        Balance b = Balance.of(bill, List.of(payment(1, PaymentType.DEPOSIT, 300_000, PaymentStatus.FAILED)), List.of());

        assertThat(b.due()).isEqualTo(vnd(1_000_000));
        assertThat(b.statusFor(bill)).isEqualTo(BillStatus.PENDING);
    }

    @Test
    void partialThenFull() {
        Bill bill = pricedBill();
        Payment deposit = payment(1, PaymentType.DEPOSIT, 300_000, PaymentStatus.SUCCEEDED);
        Payment fin = payment(2, PaymentType.FINAL, 700_000, PaymentStatus.SUCCEEDED);

        Balance b = Balance.of(bill, List.of(deposit, fin), List.of());

        assertThat(b.balance()).isEqualTo(vnd(0));
        assertThat(b.statusFor(bill)).isEqualTo(BillStatus.PAID);
    }

    @Test
    void onlyCapturedPaymentsCount() {
        Bill bill = pricedBill();
        Balance b = Balance.of(bill, List.of(
                payment(1, PaymentType.FINAL, 500_000, PaymentStatus.PROCESSING),
                payment(2, PaymentType.FINAL, 500_000, PaymentStatus.REQUIRES_PAYMENT),
                payment(3, PaymentType.FINAL, 500_000, PaymentStatus.CANCELED)), List.of());

        assertThat(b.paid()).isEqualTo(vnd(0));
        assertThat(b.due()).isEqualTo(vnd(1_000_000));
    }

    @Test
    void overpaidIsARefundDue() {
        Bill bill = pricedBill();
        Balance b = Balance.of(bill, List.of(
                payment(1, PaymentType.DEPOSIT, 300_000, PaymentStatus.SUCCEEDED),
                payment(2, PaymentType.FINAL, 1_000_000, PaymentStatus.SUCCEEDED)), List.of());

        assertThat(b.refundDue()).isEqualTo(vnd(300_000));
        assertThat(b.due()).isEqualTo(vnd(0));
        assertThat(b.statusFor(bill)).isEqualTo(BillStatus.PAID);
    }

    @Test
    void succeededRefundsReopenTheBalance() {
        Bill bill = pricedBill();
        Balance b = Balance.of(bill,
                List.of(payment(1, PaymentType.FINAL, 1_000_000, PaymentStatus.PARTIALLY_REFUNDED)),
                List.of(refund(1, 200_000, RefundStatus.SUCCEEDED), refund(1, 50_000, RefundStatus.FAILED)));

        assertThat(b.refunded()).isEqualTo(vnd(200_000));
        assertThat(b.due()).isEqualTo(vnd(200_000));
        assertThat(b.statusFor(bill)).isEqualTo(BillStatus.PARTIALLY_PAID);
    }

    @Test
    void aFullyRefundedPaymentIsNotCountedTwice() {
        // The payment is REFUNDED and its refund SUCCEEDED: the money went in
        // and came back out once, so the balance is the whole total again.
        Bill bill = pricedBill();
        Balance b = Balance.of(bill,
                List.of(payment(1, PaymentType.FINAL, 1_000_000, PaymentStatus.REFUNDED)),
                List.of(refund(1, 1_000_000, RefundStatus.SUCCEEDED)));

        assertThat(b.held()).isEqualTo(vnd(0));
        assertThat(b.due()).isEqualTo(vnd(1_000_000));
        assertThat(b.statusFor(bill)).isEqualTo(BillStatus.PENDING);
    }

    @Test
    void voidBillOwesNothingAndMoneyHeldIsARefundDue() {
        Bill bill = openBill().withStatus(BillStatus.VOID);
        Balance b = Balance.of(bill, List.of(payment(1, PaymentType.DEPOSIT, 300_000, PaymentStatus.SUCCEEDED)), List.of());

        assertThat(b.total()).isEqualTo(vnd(0));
        assertThat(b.due()).isEqualTo(vnd(0));
        assertThat(b.refundDue()).isEqualTo(vnd(300_000));
        assertThat(b.statusFor(bill)).isEqualTo(BillStatus.VOID);
    }

    @Test
    void voidBillFullyRefundedIsSettled() {
        Bill bill = pricedBill().withStatus(BillStatus.VOID);
        Balance b = Balance.of(bill,
                List.of(payment(1, PaymentType.DEPOSIT, 300_000, PaymentStatus.REFUNDED)),
                List.of(refund(1, 300_000, RefundStatus.SUCCEEDED)));

        assertThat(b.balance()).isEqualTo(vnd(0));
        assertThat(b.refundDue()).isEqualTo(vnd(0));
    }
}
