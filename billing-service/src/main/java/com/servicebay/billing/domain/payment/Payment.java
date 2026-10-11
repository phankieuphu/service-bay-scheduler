package com.servicebay.billing.domain.payment;

import com.servicebay.billing.domain.money.Money;
import java.time.Instant;
import java.util.Objects;

/**
 * @param idempotencyKey the client's Idempotency-Key (or one derived from it), also sent to Stripe
 */
public record Payment(
        Long id,
        long billId,
        PaymentType type,
        Money amount,
        PaymentStatus status,
        String stripePaymentIntentId,
        String idempotencyKey,
        String failureCode,
        Instant succeededAt,
        long version) {

    public Payment {
        Objects.requireNonNull(type, "type");
        Objects.requireNonNull(amount, "amount");
        Objects.requireNonNull(status, "status");
        Objects.requireNonNull(idempotencyKey, "idempotencyKey");
        if (!amount.isPositive()) {
            throw new IllegalArgumentException("payment amount must be positive");
        }
    }

    public static Payment requested(long billId, PaymentType type, Money amount, String idempotencyKey) {
        return new Payment(null, billId, type, amount, PaymentStatus.REQUIRES_PAYMENT, null, idempotencyKey, null, null, 0);
    }
}
