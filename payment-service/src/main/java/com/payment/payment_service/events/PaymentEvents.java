package com.payment.payment_service.events;

import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.UUID;

import com.payment.payment_service.entity.OutboxMessageEntity;
import com.payment.payment_service.entity.PaymentEntity;

import tools.jackson.databind.json.JsonMapper;

// Builds the outbox rows for payment-service's events. Same envelope as every
// other service (architecture-design.md §3): event_id, event_type, occurred_at,
// and the payload under a key named after the entity (`payment`). Key = bill id,
// so all events for one bill stay ordered on one partition.
public final class PaymentEvents {

  public static final String TOPIC_SUCCEEDED = "payment.payment.succeeded.v1";
  public static final String TOPIC_FAILED = "payment.payment.failed.v1";

  private static final JsonMapper JSON = JsonMapper.builder().build();

  private PaymentEvents() {
  }

  public static OutboxMessageEntity succeeded(PaymentEntity payment) {
    return message(TOPIC_SUCCEEDED, "PaymentSucceeded", payment);
  }

  // Terminal failure only (the intent was canceled). A declined card attempt is
  // not published: the customer can retry on the same intent.
  public static OutboxMessageEntity failed(PaymentEntity payment) {
    return message(TOPIC_FAILED, "PaymentFailed", payment);
  }

  private static OutboxMessageEntity message(String topic, String eventType, PaymentEntity payment) {
    Map<String, Object> body = new LinkedHashMap<>();
    body.put("id", payment.getId());
    body.put("bill_id", payment.getBillId());
    body.put("billing_payment_id", payment.getBillingPaymentId());
    body.put("customer_id", payment.getCustomerId());
    // A string, not a JSON number, so no consumer ever parses money as a float.
    body.put("amount", payment.getAmount().toPlainString());
    body.put("currency", payment.getCurrency());
    body.put("status", payment.getStatus().name());
    body.put("stripe_payment_intent_id", payment.getStripePaymentIntentId());
    body.put("failure_reason", payment.getFailureReason());

    Map<String, Object> envelope = new LinkedHashMap<>();
    envelope.put("event_id", UUID.randomUUID().toString());
    envelope.put("event_type", eventType);
    envelope.put("occurred_at", Instant.now().toString());
    envelope.put("payment", body);

    return new OutboxMessageEntity(topic, payment.getBillId().toString(), JSON.writeValueAsString(envelope));
  }
}
