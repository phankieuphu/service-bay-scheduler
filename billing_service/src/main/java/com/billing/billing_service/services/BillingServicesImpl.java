package com.billing.billing_service.services;

import java.math.BigDecimal;
import java.util.List;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.billing.billing_service.entity.BillStatus;
import com.billing.billing_service.entity.BillingEntity;
import com.billing.billing_service.entity.PaymentEntity;
import com.billing.billing_service.entity.PaymentType;
import com.billing.billing_service.exceptions.ConflictException;
import com.billing.billing_service.exceptions.NotFoundException;
import com.billing.billing_service.ports.BillingService;
import com.billing.billing_service.repositories.BillingRepository;
import com.billing.billing_service.repositories.PaymentRepository;

@Service
public class BillingServicesImpl implements BillingService {

  private final BillingRepository billingRepository;
  private final PaymentRepository paymentRepository;

  public BillingServicesImpl(BillingRepository billingRepository, PaymentRepository paymentRepository) {
    this.billingRepository = billingRepository;
    this.paymentRepository = paymentRepository;
  }

  @Override
  @Transactional
  public BillingEntity createBill(Long appointmentId, Long customerId, Long vehicleId,
      BigDecimal warrantyFee, BigDecimal serviceFee, BigDecimal materialCost) {
    // One bill per appointment; the UNIQUE constraint on appointment_id is the
    // real guard, this check just gives a clean 409 in the common case.
    if (billingRepository.existsByAppointmentId(appointmentId)) {
      throw new ConflictException("bill already exists for appointment " + appointmentId);
    }

    BillingEntity bill = new BillingEntity(appointmentId, customerId, vehicleId,
        orZero(warrantyFee), orZero(serviceFee), orZero(materialCost));
    return billingRepository.save(bill);
  }

  @Override
  @Transactional(readOnly = true)
  public BillingEntity getBillingEntityById(Long id) {
    return billingRepository.findById(id)
        .orElseThrow(() -> new NotFoundException("bill " + id + " not found"));
  }

  @Override
  @Transactional(readOnly = true)
  public List<BillingEntity> getBillsByCustomer(Long customerId) {
    return billingRepository.findByCustomerIdOrderByCreatedAtDesc(customerId);
  }

  @Override
  @Transactional
  public PaymentEntity recordPayment(Long billId, PaymentType type, String method, BigDecimal amount) {
    BillingEntity bill = billingRepository.findByIdForUpdate(billId)
        .orElseThrow(() -> new NotFoundException("bill " + billId + " not found"));

    if (bill.getStatus() == BillStatus.VOID || bill.getStatus() == BillStatus.PAID) {
      throw new ConflictException("bill " + billId + " is " + bill.getStatus() + " and cannot accept payments");
    }

    BigDecimal paid = paymentRepository.sumSucceededAmount(billId);
    BigDecimal outstanding = bill.getTotalAmount().subtract(paid);
    if (amount.compareTo(outstanding) > 0) {
      throw new ConflictException("payment " + amount + " exceeds outstanding balance " + outstanding);
    }

    // No payment gateway yet: payments are accepted immediately.
    PaymentEntity payment = new PaymentEntity(billId, type, method, amount);
    payment.markSucceeded();
    paymentRepository.save(payment);

    BigDecimal newPaid = paid.add(amount);
    bill.setStatus(newPaid.compareTo(bill.getTotalAmount()) >= 0 ? BillStatus.PAID : BillStatus.PARTIALLY_PAID);
    billingRepository.save(bill);

    return payment;
  }

  @Override
  @Transactional(readOnly = true)
  public List<PaymentEntity> getPayments(Long billId) {
    if (!billingRepository.existsById(billId)) {
      throw new NotFoundException("bill " + billId + " not found");
    }
    return paymentRepository.findByBillIdOrderByCreatedAtAsc(billId);
  }

  @Override
  @Transactional
  public BillingEntity voidBill(Long billId) {
    BillingEntity bill = billingRepository.findByIdForUpdate(billId)
        .orElseThrow(() -> new NotFoundException("bill " + billId + " not found"));

    // Only an untouched bill can be voided; anything with money on it needs a refund flow first.
    if (bill.getStatus() != BillStatus.PENDING) {
      throw new ConflictException("bill " + billId + " is " + bill.getStatus() + " and cannot be voided");
    }

    bill.setStatus(BillStatus.VOID);
    return billingRepository.save(bill);
  }

  private static BigDecimal orZero(BigDecimal value) {
    return value == null ? BigDecimal.ZERO : value;
  }
}
