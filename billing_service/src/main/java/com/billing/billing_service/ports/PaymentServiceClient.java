package com.billing.billing_service.ports;

import java.math.BigDecimal;

// billing-service's view of payment-service (the only caller of Stripe).
// PaymentServiceHttpClient is the implementation; tests mock this interface.
//
// Both methods throw ConflictException when payment-service answers 409 and
// PaymentGatewayException for anything else that isn't a success.
public interface PaymentServiceClient {

  // Idempotent per billingPaymentId: a retry returns the same checkout.
  CardCheckout createCardPayment(Long billId, Long billingPaymentId, Long customerId, BigDecimal amount,
      String currency);

  void cancelCardPayment(Long externalPaymentId);

  // paymentId is payment-service's id; clientSecret goes to Stripe.js in the browser.
  record CardCheckout(Long paymentId, String clientSecret) {
  }
}
