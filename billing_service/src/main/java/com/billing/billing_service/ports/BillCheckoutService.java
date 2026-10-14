package com.billing.billing_service.ports;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;

import com.billing.billing_service.entity.BillLineEntity;
import com.billing.billing_service.entity.BillingEntity;

// The service-completion workflow: validates the customer and vehicle against
// their owning services, applies the warranty rule from business-requirement.md
// §6, and issues an itemized bill. Stands in for consuming ServiceCompleted until
// scheduler-service exists (architecture-design.md §6).
public interface BillCheckoutService {

  Checkout checkout(CheckoutCommand command);

  List<BillLineEntity> getLines(Long billId);

  // serviceDate decides warranty coverage (defaults to today when null).
  record CheckoutCommand(Long appointmentId, Long customerId, Long vehicleId, LocalDate serviceDate,
      List<Line> services, List<Line> materials) {
  }

  record Line(String description, int quantity, BigDecimal unitPrice) {
  }

  record Checkout(BillingEntity bill, List<BillLineEntity> lines, VehicleDirectory.Warranty warranty) {
  }
}
