package com.billing.billing_service.entity;

public enum BillLineKind {
  // business-requirement.md §6: charged only when the vehicle's warranty has
  // expired or it has none.
  WARRANTY_FEE,
  SERVICE,
  MATERIAL
}
