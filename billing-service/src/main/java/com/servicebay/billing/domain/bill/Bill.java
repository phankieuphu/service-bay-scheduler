package com.servicebay.billing.domain.bill;

import com.servicebay.billing.domain.money.Money;
import java.time.Instant;
import java.util.List;
import java.util.Objects;

/**
 * The bill for one appointment. It's created by the first money event: a
 * deposit at booking (OPEN, unpriced) or ServiceCompleted (priced straight
 * away). Fees are copies taken when the bill is priced, never live lookups.
 */
public record Bill(
        Long id,
        long appointmentId,
        long customerId,
        long vehicleId,
        String currency,
        Money warrantyFee,
        Money serviceFee,
        Money materialCost,
        BillStatus status,
        Instant finalizedAt,
        List<BillLine> lines,
        long version) {

    public Bill {
        Objects.requireNonNull(currency, "currency");
        Objects.requireNonNull(status, "status");
        lines = List.copyOf(lines);
        for (Money fee : List.of(warrantyFee, serviceFee, materialCost)) {
            if (!fee.currency().equals(currency)) {
                throw new IllegalArgumentException("fee currency " + fee.currency() + " != bill currency " + currency);
            }
            if (fee.isNegative()) {
                throw new IllegalArgumentException("fees must not be negative");
            }
        }
    }

    /** A new, unpriced bill, as a deposit at booking creates it. */
    public static Bill open(long appointmentId, long customerId, long vehicleId, String currency) {
        Money zero = Money.zero(currency);
        return new Bill(null, appointmentId, customerId, vehicleId, zero.currency(),
                zero, zero, zero, BillStatus.OPEN, null, List.of(), 0);
    }

    public Money total() {
        return warrantyFee.plus(serviceFee).plus(materialCost);
    }

    public boolean isPriced() {
        return finalizedAt != null;
    }

    /**
     * Prices an OPEN bill from its lines: service and warranty-fee lines are
     * the service fee and warranty fee, material lines the material cost.
     * The status that follows depends on what's already been paid (see
     * {@link com.servicebay.billing.domain.balance.Balance#statusFor}).
     */
    public Bill price(List<BillLine> pricedLines, Instant at) {
        if (status != BillStatus.OPEN) {
            throw new IllegalStateException("only an OPEN bill can be priced; this one is " + status);
        }
        Money service = Money.zero(currency);
        Money material = Money.zero(currency);
        Money warranty = Money.zero(currency);
        for (BillLine line : pricedLines) {
            switch (line.kind()) {
                case SERVICE -> service = service.plus(line.total());
                case MATERIAL -> material = material.plus(line.total());
                case WARRANTY_FEE -> warranty = warranty.plus(line.total());
            }
        }
        return new Bill(id, appointmentId, customerId, vehicleId, currency,
                warranty, service, material, BillStatus.PENDING, at, pricedLines, version);
    }

    public Bill withStatus(BillStatus next) {
        if (status == BillStatus.VOID && next != BillStatus.VOID) {
            throw new IllegalStateException("a VOID bill can't change status");
        }
        return new Bill(id, appointmentId, customerId, vehicleId, currency,
                warrantyFee, serviceFee, materialCost, next, finalizedAt, lines, version);
    }
}
