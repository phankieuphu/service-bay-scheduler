package com.billing.billing_service.ports;

import java.time.LocalDate;
import java.util.Optional;
import java.util.Set;

// Read-only view of vehicle-service used by checkout. Throws
// UpstreamServiceException when vehicle-service can't answer.
public interface VehicleDirectory {

  // Empty when the vehicle doesn't exist.
  Optional<Vehicle> findVehicle(Long vehicleId);

  // Ids of the vehicles the customer owns today (no ownership history).
  Set<Long> currentVehicleIds(Long customerId);

  Warranty warranty(Long vehicleId, LocalDate asOf);

  // status: ACTIVE | SOLD | SCRAPPED.
  record Vehicle(Long id, String vin, String licensePlate, String status) {
  }

  // warrantyEndDate is null when the vehicle has no warranty on record.
  record Warranty(boolean inWarranty, String warrantyEndDate, String asOf) {
  }
}
