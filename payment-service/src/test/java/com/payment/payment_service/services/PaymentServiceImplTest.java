package com.payment.payment_service.services;

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

import com.payment.payment_service.entity.OutboxMessageEntity;
import com.payment.payment_service.entity.PaymentEntity;
import com.payment.payment_service.entity.PaymentStatus;
import com.payment.payment_service.events.PaymentEvents;
import com.payment.payment_service.exceptions.ConflictException;
import com.payment.payment_service.exceptions.GatewayException;
import com.payment.payment_service.ports.PaymentGateway;
import com.payment.payment_service.ports.PaymentGateway.GatewayIntent;
import com.payment.payment_service.ports.PaymentGateway.WebhookEvent;
import com.payment.payment_service.ports.PaymentService.Checkout;
import com.payment.payment_service.ports.PaymentService.CreatePaymentCommand;
import com.payment.payment_service.repositories.OutboxRepository;
import com.payment.payment_service.repositories.PaymentRepository;
import com.payment.payment_service.repositories.StripeEventRepository;

class PaymentServiceImplTest {

  private PaymentRepository paymentRepository;
  private StripeEventRepository stripeEventRepository;
  private OutboxRepository outboxRepository;
  private PaymentGateway gateway;
  private PaymentServiceImpl service;

  @BeforeEach
  void setUp() {
    paymentRepository = mock(PaymentRepository.class);
    stripeEventRepository = mock(StripeEventRepository.class);
    outboxRepository = mock(OutboxRepository.class);
    gateway = mock(PaymentGateway.class);
    service = new PaymentServiceImpl(paymentRepository, stripeEventRepository, outboxRepository, gateway,
        TransactionOperations.withoutTransaction());

    when(paymentRepository.save(any(PaymentEntity.class))).thenAnswer(inv -> {
      PaymentEntity p = inv.getArgument(0);
      if (p.getId() == null) {
        ReflectionTestUtils.setField(p, "id", 7L);
      }
      return p;
    });
    when(stripeEventRepository.markProcessed(anyString(), anyString())).thenReturn(1);
  }

  private static CreatePaymentCommand command(String amount) {
    return new CreatePaymentCommand(1L, 100L, 42L, new BigDecimal(amount), "USD");
  }

  private static PaymentEntity payment(PaymentStatus status, String intentId) {
    PaymentEntity p = new PaymentEntity(1L, 100L, 42L, new BigDecimal("25.00"), "usd");
    ReflectionTestUtils.setField(p, "id", 7L);
    p.setStatus(status);
    p.setStripePaymentIntentId(intentId);
    return p;
  }

  private static WebhookEvent event(String type, Long amountReceived, String failure) {
    return new WebhookEvent("evt_1", type, "pi_1", 7L, amountReceived, failure);
  }

  @Test
  void createPayment_createsIntentAndLinksIt() {
    when(paymentRepository.findByBillingPaymentId(100L)).thenReturn(Optional.empty());
    PaymentEntity stored = payment(PaymentStatus.PENDING, null);
    when(paymentRepository.findByIdForUpdate(7L)).thenReturn(Optional.of(stored));
    when(gateway.createIntent(eq(7L), eq(1L), any(), eq("usd")))
        .thenReturn(new GatewayIntent("pi_1", "pi_1_secret", "requires_payment_method", 0L, null));

    Checkout checkout = service.createPayment(command("25.00"));

    assertEquals("pi_1_secret", checkout.clientSecret());
    assertEquals("pi_1", checkout.payment().getStripePaymentIntentId());
  }

  @Test
  void createPayment_retryReusesExistingIntent() {
    when(paymentRepository.findByBillingPaymentId(100L))
        .thenReturn(Optional.of(payment(PaymentStatus.PENDING, "pi_1")));
    when(gateway.retrieveIntent("pi_1")).thenReturn(new GatewayIntent("pi_1", "pi_1_secret", "requires_payment_method", 0L, null));

    Checkout checkout = service.createPayment(command("25.00"));

    assertEquals("pi_1_secret", checkout.clientSecret());
    verify(gateway, never()).createIntent(anyLong(), anyLong(), any(), anyString());
  }

  @Test
  void createPayment_retryWithDifferentAmountIsConflict() {
    when(paymentRepository.findByBillingPaymentId(100L))
        .thenReturn(Optional.of(payment(PaymentStatus.PENDING, "pi_1")));

    assertThrows(ConflictException.class, () -> service.createPayment(command("30.00")));
    verify(gateway, never()).retrieveIntent(anyString());
  }

  @Test
  void createPayment_rejectsFractionalZeroDecimalAmount() {
    CreatePaymentCommand vnd = new CreatePaymentCommand(1L, 100L, 42L, new BigDecimal("1000.50"), "vnd");

    assertThrows(IllegalArgumentException.class, () -> service.createPayment(vnd));
    verify(paymentRepository, never()).save(any());
  }

  @Test
  void createPayment_canceledWhileCreatingCancelsTheNewIntent() {
    when(paymentRepository.findByBillingPaymentId(100L)).thenReturn(Optional.empty());
    when(paymentRepository.findByIdForUpdate(7L)).thenReturn(Optional.of(payment(PaymentStatus.CANCELED, null)));
    when(gateway.createIntent(anyLong(), anyLong(), any(), anyString()))
        .thenReturn(new GatewayIntent("pi_1", "pi_1_secret", "requires_payment_method", 0L, null));

    assertThrows(ConflictException.class, () -> service.createPayment(command("25.00")));
    verify(gateway).cancelIntent("pi_1");
  }

  @Test
  void webhookSucceeded_marksSucceededAndWritesOutboxEvent() {
    PaymentEntity p = payment(PaymentStatus.PENDING, "pi_1");
    p.setFailureReason("card declined");
    when(paymentRepository.findByIdForUpdate(7L)).thenReturn(Optional.of(p));

    service.handleWebhook(event("payment_intent.succeeded", 2500L, null));

    assertEquals(PaymentStatus.SUCCEEDED, p.getStatus());
    assertNull(p.getFailureReason());
    ArgumentCaptor<OutboxMessageEntity> outbox = ArgumentCaptor.forClass(OutboxMessageEntity.class);
    verify(outboxRepository).save(outbox.capture());
    assertEquals(PaymentEvents.TOPIC_SUCCEEDED, outbox.getValue().getTopic());
    assertEquals("1", outbox.getValue().getMessageKey());
    assertTrue(outbox.getValue().getPayload().contains("\"event_type\":\"PaymentSucceeded\""));
    assertTrue(outbox.getValue().getPayload().contains("\"amount\":\"25.00\""));
  }

  @Test
  void webhookDuplicateEventIsIgnored() {
    when(stripeEventRepository.markProcessed("evt_1", "payment_intent.succeeded")).thenReturn(0);

    service.handleWebhook(event("payment_intent.succeeded", 2500L, null));

    verify(paymentRepository, never()).findByIdForUpdate(anyLong());
    verify(outboxRepository, never()).save(any());
  }

  @Test
  void webhookFailedAttemptKeepsPaymentOpen() {
    PaymentEntity p = payment(PaymentStatus.PENDING, "pi_1");
    when(paymentRepository.findByIdForUpdate(7L)).thenReturn(Optional.of(p));

    service.handleWebhook(event("payment_intent.payment_failed", null, "Your card was declined."));

    assertEquals(PaymentStatus.PENDING, p.getStatus());
    assertEquals("Your card was declined.", p.getFailureReason());
    verify(outboxRepository, never()).save(any());
  }

  @Test
  void webhookCanceledPublishesFailure() {
    PaymentEntity p = payment(PaymentStatus.PENDING, "pi_1");
    when(paymentRepository.findByIdForUpdate(7L)).thenReturn(Optional.of(p));

    service.handleWebhook(event("payment_intent.canceled", null, "canceled: abandoned"));

    assertEquals(PaymentStatus.CANCELED, p.getStatus());
    ArgumentCaptor<OutboxMessageEntity> outbox = ArgumentCaptor.forClass(OutboxMessageEntity.class);
    verify(outboxRepository).save(outbox.capture());
    assertEquals(PaymentEvents.TOPIC_FAILED, outbox.getValue().getTopic());
  }

  @Test
  void webhookNeverMovesATerminalPayment() {
    PaymentEntity p = payment(PaymentStatus.SUCCEEDED, "pi_1");
    when(paymentRepository.findByIdForUpdate(7L)).thenReturn(Optional.of(p));

    service.handleWebhook(event("payment_intent.processing", null, null));

    assertEquals(PaymentStatus.SUCCEEDED, p.getStatus());
    verify(paymentRepository, never()).save(any());
  }

  @Test
  void webhookForUnknownIntentIsIgnored() {
    when(paymentRepository.findByStripePaymentIntentIdForUpdate("pi_other")).thenReturn(Optional.empty());

    service.handleWebhook(new WebhookEvent("evt_2", "payment_intent.succeeded", "pi_other", null, 100L, null));

    verify(outboxRepository, never()).save(any());
  }

  @Test
  void webhookOfUnrelatedTypeIsNotRecorded() {
    service.handleWebhook(new WebhookEvent("evt_3", "customer.created", null, null, null, null));

    verify(stripeEventRepository, never()).markProcessed(anyString(), anyString());
  }

  @Test
  void cancelPayment_cancelsIntentAndPublishesFailure() {
    PaymentEntity p = payment(PaymentStatus.PENDING, "pi_1");
    when(paymentRepository.findById(7L)).thenReturn(Optional.of(p));
    when(paymentRepository.findByIdForUpdate(7L)).thenReturn(Optional.of(p));

    service.cancelPayment(7L);

    verify(gateway).cancelIntent("pi_1");
    assertEquals(PaymentStatus.CANCELED, p.getStatus());
    verify(outboxRepository).save(any(OutboxMessageEntity.class));
  }

  @Test
  void cancelPayment_stripeRefusalLeavesPaymentUntouched() {
    PaymentEntity p = payment(PaymentStatus.PROCESSING, "pi_1");
    when(paymentRepository.findById(7L)).thenReturn(Optional.of(p));
    when(gateway.cancelIntent("pi_1")).thenThrow(new GatewayException("unexpected state", null, true));

    assertThrows(GatewayException.class, () -> service.cancelPayment(7L));
    assertEquals(PaymentStatus.PROCESSING, p.getStatus());
    verify(outboxRepository, never()).save(any());
  }

  @Test
  void cancelPayment_succeededIsConflict() {
    when(paymentRepository.findById(7L)).thenReturn(Optional.of(payment(PaymentStatus.SUCCEEDED, "pi_1")));

    assertThrows(ConflictException.class, () -> service.cancelPayment(7L));
    verify(gateway, never()).cancelIntent(anyString());
  }

  @Test
  void confirmTestPayment_successPublishesLikeTheWebhook() {
    PaymentEntity p = payment(PaymentStatus.PENDING, "pi_1");
    when(gateway.isTestMode()).thenReturn(true);
    when(paymentRepository.findById(7L)).thenReturn(Optional.of(p));
    when(paymentRepository.findByIdForUpdate(7L)).thenReturn(Optional.of(p));
    when(gateway.confirmIntent(eq("pi_1"), eq("pm_card_visa"), anyString()))
        .thenReturn(new GatewayIntent("pi_1", null, "succeeded", 2500L, null));

    service.confirmTestPayment(7L, null);

    assertEquals(PaymentStatus.SUCCEEDED, p.getStatus());
    ArgumentCaptor<OutboxMessageEntity> outbox = ArgumentCaptor.forClass(OutboxMessageEntity.class);
    verify(outboxRepository).save(outbox.capture());
    assertEquals(PaymentEvents.TOPIC_SUCCEEDED, outbox.getValue().getTopic());
  }

  @Test
  void confirmTestPayment_declineKeepsPaymentOpen() {
    PaymentEntity p = payment(PaymentStatus.PENDING, "pi_1");
    when(gateway.isTestMode()).thenReturn(true);
    when(paymentRepository.findById(7L)).thenReturn(Optional.of(p));
    when(paymentRepository.findByIdForUpdate(7L)).thenReturn(Optional.of(p));
    when(gateway.confirmIntent(eq("pi_1"), eq("pm_card_chargeDeclined"), anyString()))
        .thenReturn(new GatewayIntent("pi_1", null, "requires_payment_method", 0L, "Your card was declined."));

    service.confirmTestPayment(7L, "pm_card_chargeDeclined");

    assertEquals(PaymentStatus.PENDING, p.getStatus());
    assertEquals("Your card was declined.", p.getFailureReason());
    verify(outboxRepository, never()).save(any());
  }

  @Test
  void confirmTestPayment_refusedWithLiveKey() {
    when(gateway.isTestMode()).thenReturn(false);

    assertThrows(com.payment.payment_service.exceptions.TestModeOnlyException.class,
        () -> service.confirmTestPayment(7L, null));
    verify(gateway, never()).confirmIntent(anyString(), anyString(), anyString());
  }

  @Test
  void syncPayment_appliesStripeStatusAndNeverMovesTerminal() {
    PaymentEntity p = payment(PaymentStatus.PENDING, "pi_1");
    when(paymentRepository.findById(7L)).thenReturn(Optional.of(p));
    when(paymentRepository.findByIdForUpdate(7L)).thenReturn(Optional.of(p));
    when(gateway.retrieveIntent("pi_1")).thenReturn(new GatewayIntent("pi_1", null, "canceled", 0L, "canceled: abandoned"));

    service.syncPayment(7L);
    assertEquals(PaymentStatus.CANCELED, p.getStatus());

    when(gateway.retrieveIntent("pi_1")).thenReturn(new GatewayIntent("pi_1", null, "succeeded", 2500L, null));
    service.syncPayment(7L);
    assertEquals(PaymentStatus.CANCELED, p.getStatus());
    verify(outboxRepository).save(any(OutboxMessageEntity.class)); // only the cancel
  }

  @Test
  void stripeStatusMapping() {
    assertEquals(PaymentStatus.PENDING, PaymentServiceImpl.fromStripeStatus("requires_action"));
    assertEquals(PaymentStatus.PROCESSING, PaymentServiceImpl.fromStripeStatus("processing"));
    assertEquals(PaymentStatus.SUCCEEDED, PaymentServiceImpl.fromStripeStatus("succeeded"));
  }
}
