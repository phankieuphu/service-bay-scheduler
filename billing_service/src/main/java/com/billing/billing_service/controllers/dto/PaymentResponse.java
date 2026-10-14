package com.billing.billing_service.controllers.dto;

import java.math.BigDecimal;
import java.time.OffsetDateTime;

import com.billing.billing_service.entity.PaymentEntity;
import com.billing.billing_service.entity.PaymentMethod;
import com.billing.billing_service.entity.PaymentStatus;
import com.billing.billing_service.entity.PaymentType;

// clientSecret is only set when a CARD payment is created: the browser passes it
// to Stripe.js (stripe.confirmPayment) to collect and confirm the card.
public record PaymentResponse(
    Long id,
    Long billId,
    PaymentType type,
    PaymentMethod method,
    BigDecimal amount,
    PaymentStatus status,
    String failureReason,
    // payment-service's payment id (CARD only), e.g. for its test-confirm endpoint.
    Long externalPaymentId,
    String clientSecret,
    OffsetDateTime paidAt,
    OffsetDateTime createdAt) {

  public static PaymentResponse from(PaymentEntity payment) {
    return from(payment, null);
  }

  public static PaymentResponse from(PaymentEntity payment, String clientSecret) {
    return new PaymentResponse(payment.getId(), payment.getBillId(), payment.getType(), payment.getMethod(),
        payment.getAmount(), payment.getStatus(), payment.getFailureReason(), payment.getExternalPaymentId(), clientSecret,
        payment.getPaidAt(), payment.getCreatedAt());
  }
}
