package com.billing.billing_service.controllers.dto;

import java.math.BigDecimal;

import com.billing.billing_service.entity.PaymentMethod;
import com.billing.billing_service.entity.PaymentType;

public record PaymentRequest(PaymentType type, PaymentMethod method, BigDecimal amount) {
}
