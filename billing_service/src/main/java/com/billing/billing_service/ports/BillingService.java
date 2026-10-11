package com.billing.billing_service.ports;

import java.math.BigDecimal;
import java.util.List;

import com.billing.billing_service.entity.BillingEntity;
import com.billing.billing_service.entity.PaymentEntity;
import com.billing.billing_service.entity.PaymentType;

public interface BillingService {

  BillingEntity createBill(Long appointmentId, Long customerId, Long vehicleId,
      BigDecimal warrantyFee, BigDecimal serviceFee, BigDecimal materialCost);

  BillingEntity getBillingEntityById(Long id);

  List<BillingEntity> getBillsByCustomer(Long customerId);

  PaymentEntity recordPayment(Long billId, PaymentType type, String method, BigDecimal amount);

  List<PaymentEntity> getPayments(Long billId);

  BillingEntity voidBill(Long billId);
}
