package com.payment.payment_service.exceptions;

// Stripe rejected or failed a request (network error, API error, card declined
// at creation). Mapped to 502. `conflict` marks errors caused by the intent's
// state (e.g. canceling an intent that already succeeded), mapped to 409.
public class GatewayException extends RuntimeException {
  private final boolean conflict;

  public GatewayException(String message, Throwable cause, boolean conflict) {
    super(message, cause);
    this.conflict = conflict;
  }

  public boolean isConflict() {
    return conflict;
  }
}
