package com.billing.billing_service.services;

import java.math.BigDecimal;
import java.util.List;
import java.util.Optional;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionOperations;

import com.billing.billing_service.entity.BillStatus;
import com.billing.billing_service.entity.BillingEntity;
import com.billing.billing_service.entity.PaymentEntity;
import com.billing.billing_service.entity.PaymentMethod;
import com.billing.billing_service.entity.PaymentStatus;
import com.billing.billing_service.entity.PaymentType;
import com.billing.billing_service.events.BillingEvents;
import com.billing.billing_service.exceptions.ConflictException;
import com.billing.billing_service.exceptions.NotFoundException;
import com.billing.billing_service.exceptions.PaymentGatewayException;
import com.billing.billing_service.ports.BillingService;
import com.billing.billing_service.ports.PaymentServiceClient;
import com.billing.billing_service.ports.PaymentServiceClient.CardCheckout;
import com.billing.billing_service.repositories.BillingRepository;
import com.billing.billing_service.repositories.OutboxRepository;
import com.billing.billing_service.repositories.PaymentRepository;

// Every change to a bill's payments happens under the bill's row lock
// (findByIdForUpdate), so the outstanding-balance check and the status update
// never interleave. Transactions are opened explicitly (TransactionOperations)
// because the card flow must not hold that lock across the HTTP call to
// payment-service.
@Service
public class BillingServicesImpl implements BillingService {

  private static final Logger log = LoggerFactory.getLogger(BillingServicesImpl.class);

  private final BillingRepository billingRepository;
  private final PaymentRepository paymentRepository;
  private final OutboxRepository outboxRepository;
  private final PaymentServiceClient paymentServiceClient;
  private final TransactionOperations tx;
  private final String currency;

  public BillingServicesImpl(BillingRepository billingRepository, PaymentRepository paymentRepository,
      OutboxRepository outboxRepository, PaymentServiceClient paymentServiceClient, TransactionOperations tx,
      @Value("${billing.currency:usd}") String currency) {
    this.billingRepository = billingRepository;
    this.paymentRepository = paymentRepository;
    this.outboxRepository = outboxRepository;
    this.paymentServiceClient = paymentServiceClient;
    this.tx = tx;
    this.currency = currency;
  }

  @Override
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
  public BillingEntity getBillingEntityById(Long id) {
    return billingRepository.findById(id)
        .orElseThrow(() -> new NotFoundException("bill " + id + " not found"));
  }

  @Override
  public List<BillingEntity> getBillsByCustomer(Long customerId) {
    return billingRepository.findByCustomerIdOrderByCreatedAtDesc(customerId);
  }

  @Override
  public RecordedPayment recordPayment(Long billId, PaymentType type, PaymentMethod method, BigDecimal amount) {
    return switch (method) {
      case CASH -> new RecordedPayment(tx.execute(status -> recordCash(billId, type, amount)), null);
      case CARD -> startCardPayment(billId, type, amount);
    };
  }

  private PaymentEntity recordCash(Long billId, PaymentType type, BigDecimal amount) {
    BillingEntity bill = lockPayableBill(billId);
    BigDecimal paid = paymentRepository.sumSucceededAmount(billId);
    requireWithinOutstanding(bill, paid, amount);

    PaymentEntity payment = new PaymentEntity(billId, type, PaymentMethod.CASH, amount);
    payment.markSucceeded();
    paymentRepository.save(payment);

    applyPaidAmount(bill, paid.add(amount));
    outboxRepository.save(BillingEvents.paymentReceived(bill, payment));
    return payment;
  }

  private RecordedPayment startCardPayment(Long billId, PaymentType type, BigDecimal amount) {
    // 1. Reserve the amount as a PENDING payment, under the bill lock.
    PaymentEntity pending = tx.execute(status -> {
      BillingEntity bill = lockPayableBill(billId);
      requireWithinOutstanding(bill, paymentRepository.sumSucceededAmount(billId), amount);
      return paymentRepository.save(new PaymentEntity(billId, type, PaymentMethod.CARD, amount));
    });
    Long customerId = getBillingEntityById(billId).getCustomerId();

    // 2. Ask payment-service for a Stripe checkout, with no transaction open.
    CardCheckout checkout;
    try {
      checkout = paymentServiceClient.createCardPayment(billId, pending.getId(), customerId, amount, currency);
    } catch (PaymentGatewayException | ConflictException e) {
      // The browser never got a client secret, so this payment can't be completed;
      // release the reserved amount.
      tx.executeWithoutResult(status -> failIfPending(pending.getId(), "checkout failed: " + e.getMessage()));
      throw e;
    }

    // 3. Remember payment-service's id. The success event may already have done it.
    PaymentEntity linked = tx.execute(status -> {
      PaymentEntity payment = paymentRepository.findById(pending.getId()).orElseThrow();
      if (payment.getExternalPaymentId() == null) {
        payment.setExternalPaymentId(checkout.paymentId());
        paymentRepository.save(payment);
      }
      return payment;
    });
    return new RecordedPayment(linked, checkout.clientSecret());
  }

  @Override
  public List<PaymentEntity> getPayments(Long billId) {
    if (!billingRepository.existsById(billId)) {
      throw new NotFoundException("bill " + billId + " not found");
    }
    return paymentRepository.findByBillIdOrderByCreatedAtAsc(billId);
  }

  @Override
  public PaymentEntity cancelCardPayment(Long billId, Long paymentId) {
    PaymentEntity payment = paymentRepository.findById(paymentId)
        .filter(p -> p.getBillId().equals(billId))
        .orElseThrow(() -> new NotFoundException("payment " + paymentId + " not found on bill " + billId));
    if (payment.getMethod() != PaymentMethod.CARD || payment.getStatus() != PaymentStatus.PENDING) {
      throw new ConflictException("payment " + paymentId + " is " + payment.getMethod() + "/"
          + payment.getStatus() + " and cannot be canceled");
    }

    // payment-service answers 409 if Stripe already charged the card; that
    // propagates and the payment stays PENDING until the success event lands.
    if (payment.getExternalPaymentId() != null) {
      paymentServiceClient.cancelCardPayment(payment.getExternalPaymentId());
    }

    return tx.execute(status -> {
      lockBill(billId);
      return failIfPending(paymentId, "canceled");
    });
  }

  @Override
  public BillingEntity voidBill(Long billId) {
    return tx.execute(status -> {
      BillingEntity bill = lockBill(billId);

      // Only an untouched bill can be voided; anything with money on it needs a refund flow first.
      if (bill.getStatus() != BillStatus.PENDING) {
        throw new ConflictException("bill " + billId + " is " + bill.getStatus() + " and cannot be voided");
      }
      if (paymentRepository.existsByBillIdAndStatus(billId, PaymentStatus.PENDING)) {
        throw new ConflictException("bill " + billId + " has a card payment in progress; cancel it first");
      }

      bill.setStatus(BillStatus.VOID);
      return billingRepository.save(bill);
    });
  }

  @Override
  public void applyCardPaymentSucceeded(Long paymentId, Long externalPaymentId, BigDecimal amount) {
    tx.executeWithoutResult(status -> {
      Optional<PaymentEntity> found = lockPaymentWithBill(paymentId);
      if (found.isEmpty()) {
        log.warn("PaymentSucceeded for unknown billing payment {}", paymentId);
        return;
      }
      PaymentEntity payment = found.get();
      if (payment.getStatus() == PaymentStatus.SUCCEEDED) {
        return; // redelivery
      }
      if (payment.getMethod() != PaymentMethod.CARD) {
        log.warn("PaymentSucceeded for billing payment {} which is {}, ignoring", paymentId, payment.getMethod());
        return;
      }
      if (payment.getStatus() == PaymentStatus.FAILED) {
        // The card was charged after billing gave up on it. The money is real, so
        // record it; an overpayment shows up as paid > total and needs a refund.
        log.error("billing payment {} was {} but payment-service reports it succeeded; recording it",
            paymentId, payment.getFailureReason());
      }
      if (amount != null && amount.compareTo(payment.getAmount()) != 0) {
        log.error("billing payment {} expected {} but payment-service reports {}", paymentId,
            payment.getAmount(), amount);
      }

      BillingEntity bill = billingRepository.findById(payment.getBillId()).orElseThrow();
      BigDecimal paid = paymentRepository.sumSucceededAmount(bill.getId());
      payment.markSucceeded();
      if (payment.getExternalPaymentId() == null) {
        payment.setExternalPaymentId(externalPaymentId);
      }
      paymentRepository.save(payment);

      if (bill.getStatus() == BillStatus.VOID) {
        log.error("bill {} is VOID but received card payment {}; needs a refund", bill.getId(), paymentId);
      } else {
        applyPaidAmount(bill, paid.add(payment.getAmount()));
      }
      outboxRepository.save(BillingEvents.paymentReceived(bill, payment));
    });
  }

  @Override
  public void applyCardPaymentFailed(Long paymentId, Long externalPaymentId, String reason) {
    tx.executeWithoutResult(status -> {
      if (lockPaymentWithBill(paymentId).isEmpty()) {
        log.warn("PaymentFailed for unknown billing payment {}", paymentId);
        return;
      }
      failIfPending(paymentId, reason == null ? "payment failed" : reason);
    });
  }

  // Locks the payment's bill first (the same lock order as every other write),
  // then loads the payment fresh.
  private Optional<PaymentEntity> lockPaymentWithBill(Long paymentId) {
    return paymentRepository.findBillIdById(paymentId).flatMap(billId -> {
      lockBill(billId);
      return paymentRepository.findById(paymentId);
    });
  }

  private PaymentEntity failIfPending(Long paymentId, String reason) {
    PaymentEntity payment = paymentRepository.findById(paymentId).orElseThrow();
    if (payment.getStatus() == PaymentStatus.PENDING) {
      payment.markFailed(reason);
      paymentRepository.save(payment);
    }
    return payment;
  }

  private BillingEntity lockBill(Long billId) {
    return billingRepository.findByIdForUpdate(billId)
        .orElseThrow(() -> new NotFoundException("bill " + billId + " not found"));
  }

  private BillingEntity lockPayableBill(Long billId) {
    BillingEntity bill = lockBill(billId);
    if (bill.getStatus() == BillStatus.VOID || bill.getStatus() == BillStatus.PAID) {
      throw new ConflictException("bill " + billId + " is " + bill.getStatus() + " and cannot accept payments");
    }
    return bill;
  }

  // Card payments still in flight count as taken, so two concurrent checkouts
  // can't together exceed the bill.
  private void requireWithinOutstanding(BillingEntity bill, BigDecimal paid, BigDecimal amount) {
    BigDecimal outstanding = bill.getTotalAmount()
        .subtract(paid)
        .subtract(paymentRepository.sumPendingAmount(bill.getId()));
    if (amount.compareTo(outstanding) > 0) {
      throw new ConflictException("payment " + amount + " exceeds outstanding balance " + outstanding);
    }
  }

  private void applyPaidAmount(BillingEntity bill, BigDecimal paid) {
    bill.setStatus(paid.compareTo(bill.getTotalAmount()) >= 0 ? BillStatus.PAID : BillStatus.PARTIALLY_PAID);
    billingRepository.save(bill);
  }

  private static BigDecimal orZero(BigDecimal value) {
    return value == null ? BigDecimal.ZERO : value;
  }
}
