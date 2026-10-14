package com.payment.payment_service.exceptions;

// STRIPE_SECRET_KEY / STRIPE_WEBHOOK_SECRET is unset. Mapped to 503 so the rest
// of the stack can run locally without Stripe credentials.
public class GatewayNotConfiguredException extends RuntimeException {
  public GatewayNotConfiguredException(String message) {
    super(message);
  }
}
