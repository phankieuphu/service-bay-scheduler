package com.billing.billing_service.events;

import java.math.BigDecimal;

import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;

import com.billing.billing_service.ports.BillingService;

import tools.jackson.core.JacksonException;
import tools.jackson.databind.JsonNode;
import tools.jackson.databind.json.JsonMapper;

// Consumes payment-service's PaymentSucceeded / PaymentFailed events
// (architecture-design.md §3d) and settles the matching CARD payment.
// A message that can't be decoded or lacks required fields is logged and
// skipped, like the Go consumers do; a database error propagates so the
// listener container retries it.
@Component
public class PaymentResultConsumer {

  private static final Logger log = LoggerFactory.getLogger(PaymentResultConsumer.class);
  private static final JsonMapper JSON = JsonMapper.builder().build();

  private final BillingService billingService;

  public PaymentResultConsumer(BillingService billingService) {
    this.billingService = billingService;
  }

  @KafkaListener(topics = { "${billing.kafka.payment-succeeded-topic}", "${billing.kafka.payment-failed-topic}" })
  public void onMessage(ConsumerRecord<String, String> record) {
    JsonNode root;
    try {
      root = JSON.readTree(record.value());
    } catch (JacksonException | IllegalArgumentException e) {
      log.warn("skipping undecodable message on {} at offset {}", record.topic(), record.offset(), e);
      return;
    }

    String eventType = root.path("event_type").asString("");
    JsonNode payment = root.path("payment");
    Long billingPaymentId = longOrNull(payment.path("billing_payment_id"));
    Long externalPaymentId = longOrNull(payment.path("id"));
    if (billingPaymentId == null || externalPaymentId == null) {
      log.warn("skipping {} event {} without payment.id / payment.billing_payment_id", eventType,
          root.path("event_id").asString(""));
      return;
    }

    switch (eventType) {
      case "PaymentSucceeded" -> billingService.applyCardPaymentSucceeded(
          billingPaymentId, externalPaymentId, decimalOrNull(payment.path("amount")));
      case "PaymentFailed" -> billingService.applyCardPaymentFailed(
          billingPaymentId, externalPaymentId, payment.path("failure_reason").asString(null));
      default -> log.debug("ignoring event type {} on {}", eventType, record.topic());
    }
  }

  private static Long longOrNull(JsonNode node) {
    return node.isIntegralNumber() && node.canConvertToLong() ? node.asLong() : null;
  }

  private static BigDecimal decimalOrNull(JsonNode node) {
    try {
      return node.isMissingNode() || node.isNull() ? null : new BigDecimal(node.asString());
    } catch (NumberFormatException e) {
      return null;
    }
  }
}
