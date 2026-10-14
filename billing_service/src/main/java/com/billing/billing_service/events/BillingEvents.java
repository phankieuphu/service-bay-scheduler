package com.billing.billing_service.events;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.UUID;

import com.billing.billing_service.entity.BillingEntity;
import com.billing.billing_service.entity.OutboxMessageEntity;
import com.billing.billing_service.entity.PaymentEntity;

import tools.jackson.databind.json.JsonMapper;

// Builds the outbox rows for billing-service's events. Same envelope as every
// other service (architecture-design.md §3): event_id, event_type, occurred_at,
// and the payload under `payment`. Key = bill id.
public final class BillingEvents {

  // PaymentReceived (architecture-design.md §3): scheduler unblocks a deposit-gated
  // booking, notification sends a receipt, report counts revenue.
  public static final String TOPIC_PAYMENT_RECEIVED = "billing.payment.received.v1";

  private static final JsonMapper JSON = JsonMapper.builder().build();

  private BillingEvents() {
  }

  // Always 2 decimals, matching numeric(12,2): a freshly recorded payment still
  // carries the request's scale ("60"), a reloaded one the column's ("60.00").
  private static String money(BigDecimal amount) {
    return amount.setScale(2, RoundingMode.UNNECESSARY).toPlainString();
  }

  public static OutboxMessageEntity paymentReceived(BillingEntity bill, PaymentEntity payment) {
    Map<String, Object> body = new LinkedHashMap<>();
    body.put("id", payment.getId());
    body.put("bill_id", bill.getId());
    body.put("appointment_id", bill.getAppointmentId());
    body.put("customer_id", bill.getCustomerId());
    body.put("vehicle_id", bill.getVehicleId());
    body.put("type", payment.getType().name());
    body.put("method", payment.getMethod().name());
    // Strings, not JSON numbers, so no consumer ever parses money as a float.
    body.put("amount", money(payment.getAmount()));
    body.put("bill_total_amount", money(bill.getTotalAmount()));
    body.put("bill_status", bill.getStatus().name());
    body.put("paid_at", payment.getPaidAt() == null ? null : payment.getPaidAt().toInstant().toString());

    Map<String, Object> envelope = new LinkedHashMap<>();
    envelope.put("event_id", UUID.randomUUID().toString());
    envelope.put("event_type", "PaymentReceived");
    envelope.put("occurred_at", Instant.now().toString());
    envelope.put("payment", body);

    return new OutboxMessageEntity(TOPIC_PAYMENT_RECEIVED, bill.getId().toString(),
        JSON.writeValueAsString(envelope));
  }
}
