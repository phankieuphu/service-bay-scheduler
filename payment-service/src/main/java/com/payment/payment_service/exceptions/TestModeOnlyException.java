package com.payment.payment_service.exceptions;

// A test-only endpoint was called with a live Stripe key. Mapped to 403.
public class TestModeOnlyException extends RuntimeException {
  public TestModeOnlyException(String message) {
    super(message);
  }
}
