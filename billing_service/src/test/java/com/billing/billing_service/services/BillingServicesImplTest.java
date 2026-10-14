package com.billing.billing_service.services;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.math.BigDecimal;
import java.util.Optional;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.test.util.ReflectionTestUtils;
import org.springframework.transaction.support.TransactionOperations;

import com.billing.billing_service.entity.BillStatus;
import com.billing.billing_service.entity.BillingEntity;
import com.billing.billing_service.entity.OutboxMessageEntity;
import com.billing.billing_service.entity.PaymentEntity;
import com.billing.billing_service.entity.PaymentMethod;
import com.billing.billing_service.entity.PaymentStatus;
import com.billing.billing_service.entity.PaymentType;
import com.billing.billing_service.events.BillingEvents;
import com.billing.billing_service.exceptions.ConflictException;
import com.billing.billing_service.exceptions.NotFoundException;
import com.billing.billing_service.exceptions.PaymentGatewayException;
import com.billing.billing_service.ports.BillingService.RecordedPayment;
import com.billing.billing_service.ports.PaymentServiceClient;
import com.billing.billing_service.ports.PaymentServiceClient.CardCheckout;
import com.billing.billing_service.repositories.BillingRepository;
import com.billing.billing_service.repositories.OutboxRepository;
import com.billing.billing_service.repositories.PaymentRepository;

class BillingServicesImplTest {

  private BillingRepository billingRepository;
  private PaymentRepository paymentRepository;
  private OutboxRepository outboxRepository;
  private PaymentServiceClient paymentServiceClient;
  private BillingServicesImpl service;

  @BeforeEach
  void setUp() {
    billingRepository = mock(BillingRepository.class);
    paymentRepository = mock(PaymentRepository.class);
    outboxRepository = mock(OutboxRepository.class);
    paymentServiceClient = mock(PaymentServiceClient.class);
    service = new BillingServicesImpl(billingRepository, paymentRepository, outboxRepository, paymentServiceClient,
        TransactionOperations.withoutTransaction(), "usd");
    when(billingRepository.save(any(BillingEntity.class))).thenAnswer(inv -> inv.getArgument(0));
    when(paymentRepository.save(any(PaymentEntity.class))).thenAnswer(inv -> {
      PaymentEntity p = inv.getArgument(0);
      if (p.getId() == null) {
        ReflectionTestUtils.setField(p, "id", 5L);
      }
      return p;
    });
    when(paymentRepository.sumSucceededAmount(anyLong())).thenReturn(BigDecimal.ZERO);
    when(paymentRepository.sumPendingAmount(anyLong())).thenReturn(BigDecimal.ZERO);
  }

  private static BillingEntity bill(String warranty, String service, String material) {
    BillingEntity bill = new BillingEntity(10L, 20L, 30L, new BigDecimal(warranty), new BigDecimal(service),
        new BigDecimal(material));
    ReflectionTestUtils.setField(bill, "id", 1L);
    return bill;
  }

  private void givenBill(BillingEntity bill) {
    when(billingRepository.findByIdForUpdate(1L)).thenReturn(Optional.of(bill));
    when(billingRepository.findById(1L)).thenReturn(Optional.of(bill));
  }

  private static PaymentEntity cardPayment(String amount, PaymentStatus status) {
    PaymentEntity p = new PaymentEntity(1L, PaymentType.FINAL, PaymentMethod.CARD, new BigDecimal(amount));
    ReflectionTestUtils.setField(p, "id", 5L);
    ReflectionTestUtils.setField(p, "status", status);
    return p;
  }

  private void givenPayment(PaymentEntity p) {
    when(paymentRepository.findBillIdById(5L)).thenReturn(Optional.of(1L));
    when(paymentRepository.findById(5L)).thenReturn(Optional.of(p));
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
  void cash_partialThenFull_publishesPaymentReceived() {
    BillingEntity bill = bill("50.00", "100.00", "50.00"); // total 200
    givenBill(bill);

    RecordedPayment deposit = service.recordPayment(1L, PaymentType.DEPOSIT, PaymentMethod.CASH,
        new BigDecimal("60.00"));
    assertEquals(PaymentStatus.SUCCEEDED, deposit.payment().getStatus());
    assertNull(deposit.clientSecret());
    assertEquals(BillStatus.PARTIALLY_PAID, bill.getStatus());

    when(paymentRepository.sumSucceededAmount(1L)).thenReturn(new BigDecimal("60.00"));
    service.recordPayment(1L, PaymentType.FINAL, PaymentMethod.CASH, new BigDecimal("140.00"));
    assertEquals(BillStatus.PAID, bill.getStatus());

    ArgumentCaptor<OutboxMessageEntity> outbox = ArgumentCaptor.forClass(OutboxMessageEntity.class);
    verify(outboxRepository, org.mockito.Mockito.times(2)).save(outbox.capture());
    OutboxMessageEntity last = outbox.getValue();
    assertEquals(BillingEvents.TOPIC_PAYMENT_RECEIVED, last.getTopic());
    assertEquals("1", last.getMessageKey());
    assertTrue(last.getPayload().contains("\"event_type\":\"PaymentReceived\""));
    assertTrue(last.getPayload().contains("\"bill_status\":\"PAID\""));
    assertTrue(last.getPayload().contains("\"amount\":\"140.00\""));
  }

  @Test
  void cash_rejectsOverpayment() {
    givenBill(bill("0", "100.00", "0"));
    when(paymentRepository.sumSucceededAmount(1L)).thenReturn(new BigDecimal("80.00"));

    assertThrows(ConflictException.class,
        () -> service.recordPayment(1L, PaymentType.FINAL, PaymentMethod.CASH, new BigDecimal("30.00")));
    verify(paymentRepository, never()).save(any());
  }

  @Test
  void pendingCardPaymentCountsAgainstOutstanding() {
    givenBill(bill("0", "100.00", "0"));
    when(paymentRepository.sumPendingAmount(1L)).thenReturn(new BigDecimal("80.00"));

    assertThrows(ConflictException.class,
        () -> service.recordPayment(1L, PaymentType.FINAL, PaymentMethod.CASH, new BigDecimal("30.00")));
  }

  @Test
  void recordPayment_rejectsVoidBill() {
    BillingEntity bill = bill("0", "100.00", "0");
    bill.setStatus(BillStatus.VOID);
    givenBill(bill);

    assertThrows(ConflictException.class,
        () -> service.recordPayment(1L, PaymentType.FINAL, PaymentMethod.CASH, BigDecimal.TEN));
  }

  @Test
  void card_createsPendingPaymentAndReturnsClientSecret() {
    BillingEntity bill = bill("0", "100.00", "0");
    givenBill(bill);
    PaymentEntity[] saved = new PaymentEntity[1];
    when(paymentRepository.save(any(PaymentEntity.class))).thenAnswer(inv -> {
      PaymentEntity p = inv.getArgument(0);
      if (p.getId() == null) {
        ReflectionTestUtils.setField(p, "id", 5L);
      }
      saved[0] = p;
      return p;
    });
    when(paymentRepository.findById(5L)).thenAnswer(inv -> Optional.of(saved[0]));
    when(paymentServiceClient.createCardPayment(1L, 5L, 20L, new BigDecimal("40.00"), "usd"))
        .thenReturn(new CardCheckout(900L, "pi_1_secret"));

    RecordedPayment recorded = service.recordPayment(1L, PaymentType.DEPOSIT, PaymentMethod.CARD,
        new BigDecimal("40.00"));

    assertEquals("pi_1_secret", recorded.clientSecret());
    assertEquals(PaymentStatus.PENDING, recorded.payment().getStatus());
    assertEquals(900L, recorded.payment().getExternalPaymentId());
    assertEquals(BillStatus.PENDING, bill.getStatus()); // nothing is paid until the event arrives
    verify(outboxRepository, never()).save(any());
  }

  @Test
  void card_gatewayFailureReleasesTheReservedAmount() {
    givenBill(bill("0", "100.00", "0"));
    PaymentEntity[] saved = new PaymentEntity[1];
    when(paymentRepository.save(any(PaymentEntity.class))).thenAnswer(inv -> {
      PaymentEntity p = inv.getArgument(0);
      if (p.getId() == null) {
        ReflectionTestUtils.setField(p, "id", 5L);
      }
      saved[0] = p;
      return p;
    });
    when(paymentRepository.findById(5L)).thenAnswer(inv -> Optional.of(saved[0]));
    when(paymentServiceClient.createCardPayment(anyLong(), anyLong(), anyLong(), any(), anyString()))
        .thenThrow(new PaymentGatewayException("payment-service unreachable", null));

    assertThrows(PaymentGatewayException.class,
        () -> service.recordPayment(1L, PaymentType.FINAL, PaymentMethod.CARD, BigDecimal.TEN));
    assertEquals(PaymentStatus.FAILED, saved[0].getStatus());
  }

  @Test
  void cardSucceededEvent_marksPaidAndPublishes() {
    BillingEntity bill = bill("0", "100.00", "0");
    givenBill(bill);
    PaymentEntity p = cardPayment("100.00", PaymentStatus.PENDING);
    givenPayment(p);

    service.applyCardPaymentSucceeded(5L, 900L, new BigDecimal("100.00"));

    assertEquals(PaymentStatus.SUCCEEDED, p.getStatus());
    assertEquals(900L, p.getExternalPaymentId());
    assertEquals(BillStatus.PAID, bill.getStatus());
    verify(outboxRepository).save(any(OutboxMessageEntity.class));
  }

  @Test
  void cardSucceededEvent_redeliveryIsNoOp() {
    givenBill(bill("0", "100.00", "0"));
    givenPayment(cardPayment("100.00", PaymentStatus.SUCCEEDED));

    service.applyCardPaymentSucceeded(5L, 900L, new BigDecimal("100.00"));

    verify(paymentRepository, never()).save(any());
    verify(outboxRepository, never()).save(any());
  }

  @Test
  void cardSucceededEvent_unknownPaymentIsIgnored() {
    when(paymentRepository.findBillIdById(5L)).thenReturn(Optional.empty());

    service.applyCardPaymentSucceeded(5L, 900L, BigDecimal.TEN);

    verify(billingRepository, never()).findByIdForUpdate(anyLong());
  }

  @Test
  void cardFailedEvent_marksFailedOnlyWhenPending() {
    givenBill(bill("0", "100.00", "0"));
    PaymentEntity p = cardPayment("100.00", PaymentStatus.PENDING);
    givenPayment(p);

    service.applyCardPaymentFailed(5L, 900L, "canceled: abandoned");

    assertEquals(PaymentStatus.FAILED, p.getStatus());
    assertEquals("canceled: abandoned", p.getFailureReason());

    PaymentEntity succeeded = cardPayment("100.00", PaymentStatus.SUCCEEDED);
    givenPayment(succeeded);
    service.applyCardPaymentFailed(5L, 900L, "late");
    assertEquals(PaymentStatus.SUCCEEDED, succeeded.getStatus());
  }

  @Test
  void cancelCardPayment_cancelsAtPaymentServiceThenFails() {
    givenBill(bill("0", "100.00", "0"));
    PaymentEntity p = cardPayment("100.00", PaymentStatus.PENDING);
    p.setExternalPaymentId(900L);
    givenPayment(p);

    service.cancelCardPayment(1L, 5L);

    verify(paymentServiceClient).cancelCardPayment(900L);
    assertEquals(PaymentStatus.FAILED, p.getStatus());
  }

  @Test
  void cancelCardPayment_refusedWhenAlreadyCharged() {
    givenBill(bill("0", "100.00", "0"));
    PaymentEntity p = cardPayment("100.00", PaymentStatus.PENDING);
    p.setExternalPaymentId(900L);
    givenPayment(p);
    org.mockito.Mockito.doThrow(new ConflictException("already succeeded"))
        .when(paymentServiceClient).cancelCardPayment(900L);

    assertThrows(ConflictException.class, () -> service.cancelCardPayment(1L, 5L));
    assertEquals(PaymentStatus.PENDING, p.getStatus());
  }

  @Test
  void cancelCardPayment_wrongBillIsNotFound() {
    givenPayment(cardPayment("100.00", PaymentStatus.PENDING));

    assertThrows(NotFoundException.class, () -> service.cancelCardPayment(2L, 5L));
    verify(paymentServiceClient, never()).cancelCardPayment(eq(900L));
  }

  @Test
  void voidBill_onlyWhenPendingWithNoCardInFlight() {
    BillingEntity pending = bill("0", "100.00", "0");
    givenBill(pending);
    assertEquals(BillStatus.VOID, service.voidBill(1L).getStatus());

    BillingEntity partial = bill("0", "100.00", "0");
    partial.setStatus(BillStatus.PARTIALLY_PAID);
    when(billingRepository.findByIdForUpdate(2L)).thenReturn(Optional.of(partial));
    assertThrows(ConflictException.class, () -> service.voidBill(2L));

    BillingEntity withCard = bill("0", "100.00", "0");
    when(billingRepository.findByIdForUpdate(3L)).thenReturn(Optional.of(withCard));
    when(paymentRepository.existsByBillIdAndStatus(3L, PaymentStatus.PENDING)).thenReturn(true);
    assertThrows(ConflictException.class, () -> service.voidBill(3L));
  }
}
