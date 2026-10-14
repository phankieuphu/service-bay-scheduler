package com.payment.payment_service.ports;

import java.math.BigDecimal;

import com.payment.payment_service.entity.PaymentEntity;

public interface PaymentService {

  // Creates (or, on a retry with the same billingPaymentId, returns) the Stripe
  // PaymentIntent for one billing payment. clientSecret is what Stripe.js needs
  // to confirm the payment in the browser; it is null once the payment succeeded.
  Checkout createPayment(CreatePaymentCommand command);

  PaymentEntity getPayment(Long id);

  PaymentEntity cancelPayment(Long id);

  void handleWebhook(PaymentGateway.WebhookEvent event);

  // Reads the intent from Stripe and applies its status, exactly as the matching
  // webhook would. Recovers a payment whose webhook was missed.
  PaymentEntity syncPayment(Long id);

  // Test mode only: confirms the intent with a Stripe test payment method
  // (pm_card_visa, pm_card_chargeDeclined, ...) and syncs the result, so a full
  // card flow can run from Postman without a browser.
  PaymentEntity confirmTestPayment(Long id, String paymentMethod);

  record CreatePaymentCommand(Long billId, Long billingPaymentId, Long customerId, BigDecimal amount,
      String currency) {
  }

  record Checkout(PaymentEntity payment, String clientSecret) {
  }
}
