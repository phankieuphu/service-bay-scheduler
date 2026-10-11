package com.billing.billing_service.controllers;

import java.math.BigDecimal;
import java.util.List;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import com.billing.billing_service.controllers.dto.BillResponse;
import com.billing.billing_service.controllers.dto.CreateBillRequest;
import com.billing.billing_service.controllers.dto.PaymentRequest;
import com.billing.billing_service.controllers.dto.PaymentResponse;
import com.billing.billing_service.ports.BillingService;

@RestController
@RequestMapping("/api/v1/bills")
public class BillingServiceControler {
  private final BillingService billingService;

  public BillingServiceControler(BillingService billingService) {
    this.billingService = billingService;
  }

  @PostMapping
  @ResponseStatus(HttpStatus.CREATED)
  public BillResponse createBill(@RequestBody CreateBillRequest req) {
    if (req.appointmentId() == null || req.customerId() == null || req.vehicleId() == null) {
      throw new IllegalArgumentException("appointmentId, customerId and vehicleId are required");
    }
    requireNonNegative("warrantyFee", req.warrantyFee());
    requireNonNegative("serviceFee", req.serviceFee());
    requireNonNegative("materialCost", req.materialCost());

    return BillResponse.from(billingService.createBill(req.appointmentId(), req.customerId(), req.vehicleId(),
        req.warrantyFee(), req.serviceFee(), req.materialCost()));
  }

  @GetMapping("/{id}")
  public BillResponse getBill(@PathVariable Long id) {
    return BillResponse.from(billingService.getBillingEntityById(id));
  }

  @GetMapping
  public List<BillResponse> getBillsByCustomer(@RequestParam Long customerId) {
    return billingService.getBillsByCustomer(customerId).stream().map(BillResponse::from).toList();
  }

  @PostMapping("/{id}/payments")
  @ResponseStatus(HttpStatus.CREATED)
  public PaymentResponse recordPayment(@PathVariable Long id, @RequestBody PaymentRequest req) {
    if (req.type() == null || req.method() == null || req.method().isBlank()) {
      throw new IllegalArgumentException("type and method are required");
    }
    if (req.amount() == null || req.amount().signum() <= 0) {
      throw new IllegalArgumentException("amount must be greater than 0");
    }

    return PaymentResponse.from(billingService.recordPayment(id, req.type(), req.method(), req.amount()));
  }

  @GetMapping("/{id}/payments")
  public List<PaymentResponse> getPayments(@PathVariable Long id) {
    return billingService.getPayments(id).stream().map(PaymentResponse::from).toList();
  }

  @PostMapping("/{id}/void")
  public BillResponse voidBill(@PathVariable Long id) {
    return BillResponse.from(billingService.voidBill(id));
  }

  private static void requireNonNegative(String field, BigDecimal value) {
    if (value != null && value.signum() < 0) {
      throw new IllegalArgumentException(field + " must not be negative");
    }
  }
}
