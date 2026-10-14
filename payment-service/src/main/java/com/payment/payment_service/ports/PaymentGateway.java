package com.payment.payment_service.ports;

import java.math.BigDecimal;

// The seam between payment logic and Stripe. StripePaymentGateway is the only
// implementation; tests mock this interface so no Stripe types leak into them.
public interface PaymentGateway {

  // Idempotent per paymentId: calling it twice for the same payment returns the
  // same PaymentIntent instead of creating a second one.
  GatewayIntent createIntent(long paymentId, long billId, BigDecimal amount, String currency);

  GatewayIntent retrieveIntent(String intentId);

  GatewayIntent cancelIntent(String intentId);

  // Confirms the intent server-side with a saved/test payment method. A declined
  // card is not an exception: the returned intent is back in requires_payment_method
  // with failureReason set.
  GatewayIntent confirmIntent(String intentId, String paymentMethod, String returnUrl);

  // True only for Stripe test-mode keys (sk_test_...): gates the test-confirm endpoint.
  boolean isTestMode();

  // Verifies the Stripe-Signature header and extracts what the service needs.
  // Throws InvalidWebhookException when the signature or payload is bad.
  WebhookEvent parseWebhook(String payload, String signatureHeader);

  // status is Stripe's raw PaymentIntent status; amountReceived is in minor units.
  record GatewayIntent(String id, String clientSecret, String status, Long amountReceived, String failureReason) {
  }

  // paymentId is our payment id from the intent's metadata (null when the intent
  // wasn't created by this service). amountReceived is in minor units.
  record WebhookEvent(String id, String type, String intentId, Long paymentId,
      Long amountReceived, String failureReason) {
  }
}
