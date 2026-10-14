package com.billing.billing_service.exceptions;

// payment-service (or Stripe behind it) was unreachable or failed. Mapped to 502.
public class PaymentGatewayException extends RuntimeException {
  public PaymentGatewayException(String message, Throwable cause) {
    super(message, cause);
  }
}
