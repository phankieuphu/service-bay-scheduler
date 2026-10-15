package com.servicebay.billing.adapters.http;

import com.servicebay.billing.application.Bills.BillView;
import com.servicebay.billing.application.Bills.Page;
import com.servicebay.billing.application.Payments.Started;
import com.servicebay.billing.domain.bill.BillLine;
import com.servicebay.billing.domain.bill.BillStatus;
import com.servicebay.billing.domain.bill.LineKind;
import com.servicebay.billing.domain.payment.Payment;
import com.servicebay.billing.domain.payment.PaymentStatus;
import com.servicebay.billing.domain.payment.PaymentType;
import java.time.Instant;
import java.util.List;

/**
 * Response bodies. Field names become snake_case (spring.jackson
 * property-naming-strategy). Money is integer minor units plus the currency;
 * the client formats it (#87).
 */
final class Dtos {

    private Dtos() {
    }

    record LineDto(Long id, LineKind kind, Long sourceId, String name, int quantity, long unitPriceMinor,
            long lineTotalMinor) {

        static LineDto of(BillLine l) {
            return new LineDto(l.id(), l.kind(), l.sourceId(), l.name(), l.quantity(), l.unitPrice().amountMinor(),
                    l.total().amountMinor());
        }
    }

    /** What a bill may show about a payment: never more than brand and last four. */
    record PaymentDto(long id, long billId, PaymentType type, long amountMinor, String currency, PaymentStatus status,
            String failureCode, String failureMessage, String cardBrand, String cardLast4, Instant paidAt) {

        static PaymentDto of(Payment p) {
            return new PaymentDto(p.id(), p.billId(), p.type(), p.amount().amountMinor(), p.amount().currency(),
                    p.status(), p.failureCode(), p.failureMessage(), p.cardBrand(), p.cardLast4(), p.succeededAt());
        }
    }

    /**
     * total_minor and balance_due_minor are null while the bill is OPEN (not
     * priced yet), so a UI never shows an unpriced bill as "nothing due" (#87).
     */
    record BillDto(long id, long appointmentId, long customerId, long vehicleId, BillStatus status, String currency,
            List<LineDto> lines, long warrantyFeeMinor, long serviceFeeMinor, long materialCostMinor, Long totalMinor,
            List<PaymentDto> payments, long amountPaidMinor, Long balanceDueMinor, long refundDueMinor,
            Instant finalizedAt) {

        static BillDto of(BillView v) {
            boolean priced = v.bill().status() != BillStatus.OPEN;
            return new BillDto(v.bill().id(), v.bill().appointmentId(), v.bill().customerId(), v.bill().vehicleId(),
                    v.bill().status(), v.bill().currency(), v.bill().lines().stream().map(LineDto::of).toList(),
                    v.bill().warrantyFee().amountMinor(), v.bill().serviceFee().amountMinor(),
                    v.bill().materialCost().amountMinor(), priced ? v.bill().total().amountMinor() : null,
                    v.payments().stream().map(PaymentDto::of).toList(), v.balance().held().amountMinor(),
                    priced ? v.balance().due().amountMinor() : null, v.balance().refundDue().amountMinor(),
                    v.bill().finalizedAt());
        }
    }

    record BillPageDto(List<BillDto> bills, Long nextCursor, boolean hasMore) {

        static BillPageDto of(Page page) {
            return new BillPageDto(page.bills().stream().map(BillDto::of).toList(), page.nextCursor(),
                    page.nextCursor() != null);
        }
    }

    /** A started payment: the browser confirms client_secret with the Payment Element (#86, #88). */
    record StartedDto(long paymentId, long billId, PaymentType type, long amountMinor, String currency,
            PaymentStatus status, String clientSecret) {

        static StartedDto of(Started s) {
            Payment p = s.payment();
            return new StartedDto(p.id(), p.billId(), p.type(), p.amount().amountMinor(), p.amount().currency(),
                    p.status(), s.clientSecret());
        }
    }

    record TestConfirmRequest(String paymentMethod) {
    }

    record ErrorDto(String error) {
    }
}
