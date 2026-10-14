package com.payment.payment_service.entity;

// PENDING covers every Stripe state where the customer still has to act
// (requires_payment_method, requires_confirmation, requires_action). A failed
// card attempt goes back to PENDING: the customer can retry on the same intent.
public enum PaymentStatus {
  PENDING,
  PROCESSING,
  SUCCEEDED,
  CANCELED;

  public boolean isTerminal() {
    return this == SUCCEEDED || this == CANCELED;
  }
}
