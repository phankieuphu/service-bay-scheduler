package com.billing.billing_service.entity;

public enum PaymentMethod {
  // Taken at the desk: recorded as SUCCEEDED immediately.
  CASH,
  // Charged through payment-service (Stripe): PENDING until its event arrives.
  CARD
}
