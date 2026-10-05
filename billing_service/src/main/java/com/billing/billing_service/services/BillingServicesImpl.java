package com.billing.billing_service.services;

import org.springframework.stereotype.Service;

import com.billing.billing_service.entity.BillingEntity;
import com.billing.billing_service.ports.BillingService;
import com.billing.billing_service.repositories.BillingRepository;

@Service
public class BillingServicesImpl implements BillingService {

  private final BillingRepository billingRepository;

  public BillingServicesImpl(BillingRepository billingRepository) {
    this.billingRepository = billingRepository;
  }

  @Override
  public BillingEntity getBillingEntityById(Long id) {
    BillingEntity billing = this.billingRepository.findById(id);

    return billing;
  }

}
