package com.billing.billing_service.controllers.dto;

import java.math.BigDecimal;
import java.time.OffsetDateTime;

import com.billing.billing_service.entity.PaymentEntity;
import com.billing.billing_service.entity.PaymentStatus;
import com.billing.billing_service.entity.PaymentType;

public record PaymentResponse(
    Long id,
    Long billId,
    PaymentType type,
    String method,
    BigDecimal amount,
    PaymentStatus status,
    OffsetDateTime paidAt,
    OffsetDateTime createdAt) {

  public static PaymentResponse from(PaymentEntity payment) {
    return new PaymentResponse(payment.getId(), payment.getBillId(), payment.getType(), payment.getMethod(),
        payment.getAmount(), payment.getStatus(), payment.getPaidAt(), payment.getCreatedAt());
  }
}
