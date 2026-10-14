package com.servicebay.billing.domain.ports;

import com.servicebay.billing.domain.money.Money;
import java.util.Map;

/**
 * The payment provider (#83). StripePaymentProvider is the only
 * implementation and the only code that imports stripe-java; another provider
 * (VNPay, MoMo, ...) would be another implementation of this.
 *
 * <p>Every method throws {@link PaymentProviderException} on failure.
 */
public interface PaymentProvider {

    /** The provider's customer for one of ours, created on first use. */
    String createCustomer(long customerId, String idempotencyKey);

    /**
     * Creates an intent to collect amount. idempotencyKey is sent to the
     * provider, so a retry returns the original intent instead of a second one.
     */
    ProviderIntent createIntent(Money amount, String providerCustomerId, Map<String, String> metadata,
            String idempotencyKey);

    /** The intent's current state, read from the provider (the source of truth). */
    ProviderIntent retrieveIntent(String intentId);

    ProviderIntent cancelIntent(String intentId);

    /**
     * Test mode only: confirms with a test payment method (pm_card_visa,
     * pm_card_chargeDeclined, ...). A decline is not an exception: the
     * returned intent carries the failure.
     */
    ProviderIntent confirmTestIntent(String intentId, String paymentMethod);

    /** True only with test-mode keys; gates {@link #confirmTestIntent}. */
    boolean isTestMode();

    /**
     * Verifies the webhook signature over the raw body and returns the event.
     * Throws {@link PaymentProviderException} with kind INVALID_WEBHOOK.
     */
    ProviderEvent parseWebhook(String payload, String signatureHeader);

    /**
     * An intent as the provider sees it.
     *
     * @param status the provider's raw status (requires_payment_method, processing, succeeded, canceled, ...)
     * @param amountReceived minor units actually captured
     * @param failureCode / failureMessage the last attempt's error, if any
     * @param cardBrand / cardLast4 of the charge that succeeded, if any
     * @param paymentId our payment id from the intent's metadata; null if it isn't one of ours
     */
    record ProviderIntent(String id, String clientSecret, String status, long amount, long amountReceived,
            String failureCode, String failureMessage, String cardBrand, String cardLast4, Long paymentId) {
    }

    /**
     * A webhook event. intentId is set for payment_intent.* events; other
     * types (charge.refunded, charge.dispute.created) are acknowledged and
     * left to the refund work (#89).
     */
    record ProviderEvent(String id, String type, String intentId) {
    }
}
