package com.billing.billing_service.events;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;

import java.math.BigDecimal;

import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import com.billing.billing_service.ports.BillingService;

class PaymentResultConsumerTest {

  private BillingService billingService;
  private PaymentResultConsumer consumer;

  @BeforeEach
  void setUp() {
    billingService = mock(BillingService.class);
    consumer = new PaymentResultConsumer(billingService);
  }

  private static ConsumerRecord<String, String> record(String value) {
    return new ConsumerRecord<>("payment.payment.succeeded.v1", 0, 0L, "1", value);
  }

  @Test
  void succeededEventSettlesThePayment() {
    consumer.onMessage(record("""
        {"event_id":"e1","event_type":"PaymentSucceeded","occurred_at":"2026-10-15T10:00:00Z",
         "payment":{"id":900,"bill_id":1,"billing_payment_id":5,"amount":"40.00","currency":"usd"}}
        """));

    verify(billingService).applyCardPaymentSucceeded(5L, 900L, new BigDecimal("40.00"));
  }

  @Test
  void failedEventPassesTheReason() {
    consumer.onMessage(record("""
        {"event_id":"e2","event_type":"PaymentFailed",
         "payment":{"id":900,"billing_payment_id":5,"failure_reason":"canceled: abandoned"}}
        """));

    verify(billingService).applyCardPaymentFailed(5L, 900L, "canceled: abandoned");
  }

  @Test
  void undecodableOrIncompleteMessagesAreSkipped() {
    consumer.onMessage(record("not json"));
    consumer.onMessage(record("{\"event_type\":\"PaymentSucceeded\",\"payment\":{\"id\":900}}"));
    consumer.onMessage(record("{\"event_type\":\"PaymentSucceeded\",\"payment\":{\"id\":\"x\",\"billing_payment_id\":5}}"));

    verifyNoInteractions(billingService);
  }

  @Test
  void unknownEventTypesAreIgnored() {
    consumer.onMessage(record("{\"event_type\":\"PaymentRefunded\",\"payment\":{\"id\":900,\"billing_payment_id\":5}}"));

    verify(billingService, org.mockito.Mockito.never()).applyCardPaymentSucceeded(anyLong(), anyLong(), any());
  }
}
