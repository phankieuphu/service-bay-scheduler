package com.billing.billing_service.services;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.math.BigDecimal;
import java.util.Optional;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import com.billing.billing_service.entity.BillStatus;
import com.billing.billing_service.entity.BillingEntity;
import com.billing.billing_service.entity.PaymentEntity;
import com.billing.billing_service.entity.PaymentStatus;
import com.billing.billing_service.entity.PaymentType;
import com.billing.billing_service.exceptions.ConflictException;
import com.billing.billing_service.exceptions.NotFoundException;
import com.billing.billing_service.repositories.BillingRepository;
import com.billing.billing_service.repositories.PaymentRepository;

class BillingServicesImplTest {

  private BillingRepository billingRepository;
  private PaymentRepository paymentRepository;
  private BillingServicesImpl service;

  @BeforeEach
  void setUp() {
    billingRepository = mock(BillingRepository.class);
    paymentRepository = mock(PaymentRepository.class);
    service = new BillingServicesImpl(billingRepository, paymentRepository);
    when(billingRepository.save(any(BillingEntity.class))).thenAnswer(inv -> inv.getArgument(0));
    when(paymentRepository.save(any(PaymentEntity.class))).thenAnswer(inv -> inv.getArgument(0));
  }

  private static BillingEntity bill(String warranty, String service, String material) {
    return new BillingEntity(10L, 20L, 30L, new BigDecimal(warranty), new BigDecimal(service), new BigDecimal(material));
  }

  @Test
  void createBill_defaultsNullFeesToZero() {
    BillingEntity created = service.createBill(10L, 20L, 30L, null, new BigDecimal("100.00"), null);

    assertEquals(BillStatus.PENDING, created.getStatus());
    assertEquals(0, created.getTotalAmount().compareTo(new BigDecimal("100.00")));
  }

  @Test
  void createBill_rejectsDuplicateAppointment() {
    when(billingRepository.existsByAppointmentId(10L)).thenReturn(true);

    assertThrows(ConflictException.class,
        () -> service.createBill(10L, 20L, 30L, BigDecimal.ZERO, BigDecimal.ONE, BigDecimal.ZERO));
    verify(billingRepository, never()).save(any());
  }

  @Test
  void getBill_notFound() {
    when(billingRepository.findById(1L)).thenReturn(Optional.empty());

    assertThrows(NotFoundException.class, () -> service.getBillingEntityById(1L));
  }

  @Test
  void recordPayment_partialThenFull() {
    BillingEntity bill = bill("50.00", "100.00", "50.00"); // total 200
    when(billingRepository.findByIdForUpdate(1L)).thenReturn(Optional.of(bill));

    when(paymentRepository.sumSucceededAmount(1L)).thenReturn(BigDecimal.ZERO);
    PaymentEntity deposit = service.recordPayment(1L, PaymentType.DEPOSIT, "CARD", new BigDecimal("60.00"));
    assertEquals(PaymentStatus.SUCCEEDED, deposit.getStatus());
    assertEquals(BillStatus.PARTIALLY_PAID, bill.getStatus());

    when(paymentRepository.sumSucceededAmount(1L)).thenReturn(new BigDecimal("60.00"));
    service.recordPayment(1L, PaymentType.FINAL, "CARD", new BigDecimal("140.00"));
    assertEquals(BillStatus.PAID, bill.getStatus());
  }

  @Test
  void recordPayment_rejectsOverpayment() {
    BillingEntity bill = bill("0", "100.00", "0");
    when(billingRepository.findByIdForUpdate(1L)).thenReturn(Optional.of(bill));
    when(paymentRepository.sumSucceededAmount(1L)).thenReturn(new BigDecimal("80.00"));

    assertThrows(ConflictException.class,
        () -> service.recordPayment(1L, PaymentType.FINAL, "CASH", new BigDecimal("30.00")));
    verify(paymentRepository, never()).save(any());
  }

  @Test
  void recordPayment_rejectsVoidBill() {
    BillingEntity bill = bill("0", "100.00", "0");
    bill.setStatus(BillStatus.VOID);
    when(billingRepository.findByIdForUpdate(1L)).thenReturn(Optional.of(bill));

    assertThrows(ConflictException.class,
        () -> service.recordPayment(1L, PaymentType.FINAL, "CASH", BigDecimal.TEN));
  }

  @Test
  void voidBill_onlyWhenPending() {
    BillingEntity pending = bill("0", "100.00", "0");
    when(billingRepository.findByIdForUpdate(1L)).thenReturn(Optional.of(pending));
    assertEquals(BillStatus.VOID, service.voidBill(1L).getStatus());

    BillingEntity partial = bill("0", "100.00", "0");
    partial.setStatus(BillStatus.PARTIALLY_PAID);
    when(billingRepository.findByIdForUpdate(2L)).thenReturn(Optional.of(partial));
    assertThrows(ConflictException.class, () -> service.voidBill(2L));
  }
}
