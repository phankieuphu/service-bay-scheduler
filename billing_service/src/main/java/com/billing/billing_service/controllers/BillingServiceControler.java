package com.billing.billing_service.controllers;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import com.billing.billing_service.ports.BillingService;

@RestController
public class BillingServiceControler {
  private final BillingService billingService;

  public BillingServiceControler(BillingService billingService) {
    this.billingService = billingService;
  }

  @GetMapping("billing")
  public String getMethodName(@RequestParam String param) {
    return this.billingService.getBillingEntityById(Long.parseLong(param)).toString();
  }

}
