package com.payment.payment_service.controllers.dto;

import java.math.BigDecimal;
import java.time.OffsetDateTime;

import com.payment.payment_service.entity.PaymentEntity;
import com.payment.payment_service.entity.PaymentStatus;

// clientSecret is only set on the create response; GET never returns it.
public record PaymentResponse(
    Long id,
    Long billId,
    Long billingPaymentId,
    Long customerId,
    BigDecimal amount,
    String currency,
    PaymentStatus status,
    String stripePaymentIntentId,
    String failureReason,
    String clientSecret,
    OffsetDateTime createdAt,
    OffsetDateTime updatedAt) {

  public static PaymentResponse from(PaymentEntity payment) {
    return from(payment, null);
  }

  public static PaymentResponse from(PaymentEntity payment, String clientSecret) {
    return new PaymentResponse(payment.getId(), payment.getBillId(), payment.getBillingPaymentId(),
        payment.getCustomerId(), payment.getAmount(), payment.getCurrency(), payment.getStatus(),
        payment.getStripePaymentIntentId(), payment.getFailureReason(), clientSecret,
        payment.getCreatedAt(), payment.getUpdatedAt());
  }
}
