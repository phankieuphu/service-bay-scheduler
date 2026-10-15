package com.servicebay.billing.domain.payment;

import com.servicebay.billing.domain.money.Money;
import java.time.Instant;
import java.util.Objects;

/**
 * One attempt to collect money for a bill, backed by one Stripe PaymentIntent.
 * Webhooks (or a re-read of the intent) move its status; the transition
 * methods below never move a payment out of a final state.
 *
 * @param idempotencyKey the client's Idempotency-Key (or one derived from it), also sent to Stripe
 * @param failureCode Stripe's decline/error code, for logs and metrics
 * @param failureMessage Stripe's customer-facing message, shown by the UI
 * @param cardBrand and cardLast4: what the bill may show about the card, never more
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
        String failureMessage,
        String cardBrand,
        String cardLast4,
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
        return new Payment(null, billId, type, amount, PaymentStatus.REQUIRES_PAYMENT, null, idempotencyKey,
                null, null, null, null, null, 0);
    }

    /** No further provider event can change it (refunds are tracked separately). */
    public boolean isFinal() {
        return status == PaymentStatus.SUCCEEDED || status == PaymentStatus.CANCELED
                || status == PaymentStatus.REFUNDED || status == PaymentStatus.PARTIALLY_REFUNDED;
    }

    /** Still waiting for the customer or the card network. */
    public boolean isOpen() {
        return status == PaymentStatus.REQUIRES_PAYMENT || status == PaymentStatus.PROCESSING;
    }

    public Payment withIntent(String paymentIntentId) {
        return new Payment(id, billId, type, amount, status, paymentIntentId, idempotencyKey,
                failureCode, failureMessage, cardBrand, cardLast4, succeededAt, version);
    }

    public Payment processing() {
        return new Payment(id, billId, type, amount, PaymentStatus.PROCESSING, stripePaymentIntentId, idempotencyKey,
                null, null, cardBrand, cardLast4, succeededAt, version);
    }

    /**
     * A declined attempt. The PaymentIntent stays usable, so the payment goes
     * back to REQUIRES_PAYMENT with the reason; the customer can retry.
     */
    public Payment declined(String code, String message) {
        return new Payment(id, billId, type, amount, PaymentStatus.REQUIRES_PAYMENT, stripePaymentIntentId,
                idempotencyKey, code, message, cardBrand, cardLast4, succeededAt, version);
    }

    public Payment succeeded(Instant at, String brand, String last4) {
        return new Payment(id, billId, type, amount, PaymentStatus.SUCCEEDED, stripePaymentIntentId, idempotencyKey,
                null, null, brand, last4, at, version);
    }

    /** The intent was canceled (by us or in Stripe), or creating it failed: nothing can be paid on it. */
    public Payment canceled(String code, String message) {
        return new Payment(id, billId, type, amount, PaymentStatus.CANCELED, stripePaymentIntentId, idempotencyKey,
                code, message, cardBrand, cardLast4, succeededAt, version);
    }

    /** Creating the intent failed: there was never anything to pay. */
    public Payment failed(String code, String message) {
        return new Payment(id, billId, type, amount, PaymentStatus.FAILED, stripePaymentIntentId, idempotencyKey,
                code, message, cardBrand, cardLast4, succeededAt, version);
    }
}
