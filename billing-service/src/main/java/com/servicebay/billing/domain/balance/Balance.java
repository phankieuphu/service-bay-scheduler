package com.servicebay.billing.domain.balance;

import com.servicebay.billing.domain.bill.Bill;
import com.servicebay.billing.domain.bill.BillStatus;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.payment.Payment;
import com.servicebay.billing.domain.payment.Refund;
import com.servicebay.billing.domain.payment.RefundStatus;
import java.util.List;

/**
 * Where a bill stands: {@code balance = total - paid + refunded}, where paid
 * counts every captured payment (including ones later refunded) and refunded
 * counts succeeded refunds. Positive means the customer owes money, negative
 * means they're owed it.
 *
 * <p>An OPEN bill isn't priced, so its total counts as zero and a deposit
 * shows up as credit. A VOID bill owes nothing, so anything still held is a
 * refund due.
 *
 * @param total what the bill charges (zero while OPEN or once VOID)
 * @param paid captured payments
 * @param refunded succeeded refunds
 */
public record Balance(Money total, Money paid, Money refunded) {

    public static Balance of(Bill bill, List<Payment> payments, List<Refund> refunds) {
        String currency = bill.currency();
        Money total = (bill.status() == BillStatus.OPEN || bill.status() == BillStatus.VOID)
                ? Money.zero(currency)
                : bill.total();
        Money paid = Money.zero(currency);
        for (Payment p : payments) {
            if (bill.id() != null && p.billId() != bill.id()) {
                throw new IllegalArgumentException("payment " + p.id() + " belongs to bill " + p.billId());
            }
            if (p.status().isCaptured()) {
                paid = paid.plus(p.amount());
            }
        }
        Money refunded = Money.zero(currency);
        for (Refund r : refunds) {
            if (r.status() == RefundStatus.SUCCEEDED) {
                refunded = refunded.plus(r.amount());
            }
        }
        return new Balance(total, paid, refunded);
    }

    /** Net money held: paid minus refunded. */
    public Money held() {
        return paid.minus(refunded);
    }

    /** total - paid + refunded: positive is owed by the customer, negative is owed to them. */
    public Money balance() {
        return total.minus(held());
    }

    /** What the customer still owes; zero if nothing. */
    public Money due() {
        Money b = balance();
        return b.isPositive() ? b : Money.zero(b.currency());
    }

    /** What the customer is owed back (overpaid, or a VOID bill with money held); zero if nothing. */
    public Money refundDue() {
        Money b = balance();
        return b.isNegative() ? Money.zero(b.currency()).minus(b) : Money.zero(b.currency());
    }

    /**
     * The status a priced bill should have for this balance. OPEN and VOID
     * aren't derived from money, so they're returned as they are.
     */
    public BillStatus statusFor(Bill bill) {
        if (bill.status() == BillStatus.OPEN || bill.status() == BillStatus.VOID) {
            return bill.status();
        }
        if (!balance().isPositive()) {
            return BillStatus.PAID;
        }
        // Once the bill is priced, a deposit counts toward it like any payment.
        return held().isPositive() ? BillStatus.PARTIALLY_PAID : BillStatus.PENDING;
    }
}
