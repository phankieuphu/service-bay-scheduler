package com.billing.billing_service.controllers.dto;

import java.math.BigDecimal;

import com.billing.billing_service.entity.BillLineEntity;
import com.billing.billing_service.entity.BillLineKind;

public record BillLineResponse(Long id, BillLineKind kind, String description, int quantity,
    BigDecimal unitPrice, BigDecimal amount) {

  public static BillLineResponse from(BillLineEntity line) {
    return new BillLineResponse(line.getId(), line.getKind(), line.getDescription(), line.getQuantity(),
        line.getUnitPrice(), line.getAmount());
  }
}
