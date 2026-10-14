package com.payment.payment_service.controllers.dto;

import java.math.BigDecimal;

public record CreatePaymentRequest(
    Long billId,
    Long billingPaymentId,
    Long customerId,
    BigDecimal amount,
    String currency) {
}
