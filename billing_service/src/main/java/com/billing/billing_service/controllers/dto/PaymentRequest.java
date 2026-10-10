package com.billing.billing_service.controllers.dto;

import java.math.BigDecimal;

import com.billing.billing_service.entity.PaymentType;

public record PaymentRequest(PaymentType type, String method, BigDecimal amount) {
}
