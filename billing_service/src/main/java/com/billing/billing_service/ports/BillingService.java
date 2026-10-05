package com.billing.billing_service.ports;

import com.billing.billing_service.entity.BillingEntity;

public interface BillingService {
  BillingEntity getBillingEntityById(Long id);
}
