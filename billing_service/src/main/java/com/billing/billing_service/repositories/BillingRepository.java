package com.billing.billing_service.repositories;

import org.springframework.stereotype.Repository;

import com.billing.billing_service.entity.BillingEntity;

@Repository("billing-repository")
public class BillingRepository {
  public BillingEntity findById(Long id) {
    return new BillingEntity();
  }
}
