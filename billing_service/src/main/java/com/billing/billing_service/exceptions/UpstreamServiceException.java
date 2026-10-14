package com.billing.billing_service.exceptions;

// customer-service or vehicle-service was unreachable or answered with an
// unexpected error during checkout. Mapped to 502.
public class UpstreamServiceException extends RuntimeException {
  public UpstreamServiceException(String message, Throwable cause) {
    super(message, cause);
  }
}
