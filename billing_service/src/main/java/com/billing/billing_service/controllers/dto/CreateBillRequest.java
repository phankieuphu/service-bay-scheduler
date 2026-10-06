package com.billing.billing_service.controllers.dto;

import java.math.BigDecimal;

public record CreateBillRequest(
    Long appointmentId,
    Long customerId,
    Long vehicleId,
    BigDecimal warrantyFee,
    BigDecimal serviceFee,
    BigDecimal materialCost) {
}
