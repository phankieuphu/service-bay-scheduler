package com.billing.billing_service.controllers.dto;

import java.math.BigDecimal;
import java.time.OffsetDateTime;

import com.billing.billing_service.entity.BillStatus;
import com.billing.billing_service.entity.BillingEntity;

public record BillResponse(
    Long id,
    Long appointmentId,
    Long customerId,
    Long vehicleId,
    BigDecimal warrantyFee,
    BigDecimal serviceFee,
    BigDecimal materialCost,
    BigDecimal totalAmount,
    BillStatus status,
    OffsetDateTime createdAt,
    OffsetDateTime updatedAt) {

  public static BillResponse from(BillingEntity bill) {
    return new BillResponse(bill.getId(), bill.getAppointmentId(), bill.getCustomerId(), bill.getVehicleId(),
        bill.getWarrantyFee(), bill.getServiceFee(), bill.getMaterialCost(), bill.getTotalAmount(),
        bill.getStatus(), bill.getCreatedAt(), bill.getUpdatedAt());
  }
}
