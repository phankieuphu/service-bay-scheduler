package com.billing.billing_service.ports;

import java.util.Optional;

// Read-only view of customer-service used by checkout. Throws
// UpstreamServiceException when customer-service can't answer.
public interface CustomerDirectory {

  // Empty when the customer doesn't exist (or was deleted).
  Optional<Customer> findCustomer(Long customerId);

  // status: ACTIVE | INACTIVE | BANNED (customer-service constants/status.go).
  record Customer(Long id, String name, String status) {
  }
}
