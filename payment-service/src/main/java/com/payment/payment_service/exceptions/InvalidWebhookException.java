package com.payment.payment_service.exceptions;

public class InvalidWebhookException extends RuntimeException {
  public InvalidWebhookException(String message, Throwable cause) {
    super(message, cause);
  }
}
