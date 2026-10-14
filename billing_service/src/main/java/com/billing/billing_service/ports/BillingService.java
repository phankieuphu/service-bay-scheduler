package com.billing.billing_service.ports;

import java.math.BigDecimal;
import java.util.List;

import com.billing.billing_service.entity.BillingEntity;
import com.billing.billing_service.entity.PaymentEntity;
import com.billing.billing_service.entity.PaymentMethod;
import com.billing.billing_service.entity.PaymentType;

public interface BillingService {

  BillingEntity createBill(Long appointmentId, Long customerId, Long vehicleId,
      BigDecimal warrantyFee, BigDecimal serviceFee, BigDecimal materialCost);

  BillingEntity getBillingEntityById(Long id);

  List<BillingEntity> getBillsByCustomer(Long customerId);

  // CASH is recorded as SUCCEEDED at once. CARD creates a PENDING payment and a
  // Stripe checkout via payment-service; clientSecret is set only for CARD.
  RecordedPayment recordPayment(Long billId, PaymentType type, PaymentMethod method, BigDecimal amount);

  List<PaymentEntity> getPayments(Long billId);

  PaymentEntity cancelCardPayment(Long billId, Long paymentId);

  BillingEntity voidBill(Long billId);

  // Inbound from payment-service's events. Idempotent: a redelivered event is a no-op.
  void applyCardPaymentSucceeded(Long paymentId, Long externalPaymentId, BigDecimal amount);

  void applyCardPaymentFailed(Long paymentId, Long externalPaymentId, String reason);

  record RecordedPayment(PaymentEntity payment, String clientSecret) {
  }
}
