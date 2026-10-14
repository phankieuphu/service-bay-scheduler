package com.billing.billing_service.controllers.dto;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;

// serviceDate (YYYY-MM-DD) is optional; it decides warranty coverage and
// defaults to today in billing.time-zone.
public record CheckoutRequest(
    Long appointmentId,
    Long customerId,
    Long vehicleId,
    LocalDate serviceDate,
    List<LineRequest> services,
    List<LineRequest> materials) {

  // quantity defaults to 1.
  public record LineRequest(String description, Integer quantity, BigDecimal unitPrice) {
  }
}
