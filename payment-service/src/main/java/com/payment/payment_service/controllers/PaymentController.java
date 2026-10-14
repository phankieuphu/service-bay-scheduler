package com.payment.payment_service.controllers;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import com.payment.payment_service.controllers.dto.CreatePaymentRequest;
import com.payment.payment_service.controllers.dto.PaymentResponse;
import com.payment.payment_service.controllers.dto.TestConfirmRequest;
import com.payment.payment_service.ports.PaymentService;
import com.payment.payment_service.ports.PaymentService.Checkout;
import com.payment.payment_service.ports.PaymentService.CreatePaymentCommand;

// Internal API, called by billing-service only. The amount comes from billing's
// bill, never from the browser; the browser only ever sees the clientSecret.
@RestController
@RequestMapping("/api/v1/payments")
public class PaymentController {
  private final PaymentService paymentService;

  public PaymentController(PaymentService paymentService) {
    this.paymentService = paymentService;
  }

  // 201 on every call, including an idempotent retry for the same billingPaymentId.
  @PostMapping
  @ResponseStatus(HttpStatus.CREATED)
  public PaymentResponse createPayment(@RequestBody CreatePaymentRequest req) {
    if (req.billId() == null || req.billingPaymentId() == null || req.customerId() == null) {
      throw new IllegalArgumentException("billId, billingPaymentId and customerId are required");
    }
    if (req.amount() == null || req.amount().signum() <= 0) {
      throw new IllegalArgumentException("amount must be greater than 0");
    }

    Checkout checkout = paymentService.createPayment(new CreatePaymentCommand(
        req.billId(), req.billingPaymentId(), req.customerId(), req.amount(), req.currency()));
    return PaymentResponse.from(checkout.payment(), checkout.clientSecret());
  }

  @GetMapping("/{id}")
  public PaymentResponse getPayment(@PathVariable Long id) {
    return PaymentResponse.from(paymentService.getPayment(id));
  }

  @PostMapping("/{id}/cancel")
  public PaymentResponse cancelPayment(@PathVariable Long id) {
    return PaymentResponse.from(paymentService.cancelPayment(id));
  }

  // Pulls the intent's current state from Stripe; use when a webhook was missed.
  @PostMapping("/{id}/sync")
  public PaymentResponse syncPayment(@PathVariable Long id) {
    return PaymentResponse.from(paymentService.syncPayment(id));
  }

  // Test-mode keys only (403 otherwise). Body optional: {"paymentMethod": "pm_card_visa"}
  // by default, "pm_card_chargeDeclined" to simulate a decline.
  @PostMapping("/{id}/test-confirm")
  public PaymentResponse confirmTestPayment(@PathVariable Long id,
      @RequestBody(required = false) TestConfirmRequest req) {
    return PaymentResponse.from(paymentService.confirmTestPayment(id, req == null ? null : req.paymentMethod()));
  }
}
