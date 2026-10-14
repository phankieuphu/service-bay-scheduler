package com.payment.payment_service.controllers;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

import com.payment.payment_service.ports.PaymentGateway;
import com.payment.payment_service.ports.PaymentService;

// Public endpoint Stripe posts events to. The body is taken as a raw String
// because the signature is computed over the exact bytes Stripe sent.
// Responses: 200 handled (or deliberately ignored), 400 bad signature, 5xx any
// other failure — which makes Stripe retry with backoff for up to 3 days.
@RestController
public class StripeWebhookController {
  private final PaymentGateway gateway;
  private final PaymentService paymentService;

  public StripeWebhookController(PaymentGateway gateway, PaymentService paymentService) {
    this.gateway = gateway;
    this.paymentService = paymentService;
  }

  @PostMapping("/api/v1/webhooks/stripe")
  public ResponseEntity<Void> handle(@RequestBody String payload,
      @RequestHeader(name = "Stripe-Signature", required = false) String signature) {
    paymentService.handleWebhook(gateway.parseWebhook(payload, signature));
    return ResponseEntity.ok().build();
  }
}
