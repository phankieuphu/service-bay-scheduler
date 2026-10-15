package com.servicebay.billing.application;

import com.servicebay.billing.domain.balance.Balance;
import com.servicebay.billing.domain.bill.Bill;
import com.servicebay.billing.domain.bill.BillStatus;
import com.servicebay.billing.domain.outbox.OutboxMessage;
import com.servicebay.billing.domain.payment.Payment;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.UUID;
import tools.jackson.databind.json.JsonMapper;

/**
 * Billing's events (#84, architecture-design §3e). Same envelope as every
 * other service: event_id, event_type, occurred_at, and the payload under a
 * key named after the entity. Keyed by appointment id, so a booking's billing
 * events stay in order. Amounts are integer minor units plus currency.
 */
final class BillingEvents {

    static final String TOPIC_BILL_READY = "billing.bill.ready.v1";
    static final String TOPIC_PAYMENT_RECEIVED = "billing.payment.received.v1";
    static final String TOPIC_PAYMENT_FAILED = "billing.payment.failed.v1";

    private static final JsonMapper JSON = JsonMapper.builder().build();

    private BillingEvents() {
    }

    /** The bill is priced and a balance is due. */
    static OutboxMessage billReady(Bill bill, Balance balance) {
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("id", bill.id());
        body.put("appointment_id", bill.appointmentId());
        body.put("customer_id", bill.customerId());
        body.put("vehicle_id", bill.vehicleId());
        body.put("status", bill.status().name());
        body.put("currency", bill.currency());
        body.put("total_minor", bill.total().amountMinor());
        body.put("amount_paid_minor", balance.held().amountMinor());
        body.put("balance_due_minor", balance.due().amountMinor());
        body.put("finalized_at", bill.finalizedAt() == null ? null : bill.finalizedAt().toString());
        return message(TOPIC_BILL_READY, "BillReady", bill.appointmentId(), "bill", body);
    }

    /** A payment succeeded: scheduler confirms a booking on a DEPOSIT, notification sends a receipt. */
    static OutboxMessage paymentReceived(Bill bill, Payment payment, Balance balance) {
        Map<String, Object> body = paymentBody(bill, payment);
        body.put("bill_status", bill.status().name());
        // An OPEN bill isn't priced, so there's no balance to state yet.
        body.put("balance_due_minor", bill.status() == BillStatus.OPEN ? null : balance.due().amountMinor());
        body.put("paid_at", payment.succeededAt() == null ? null : payment.succeededAt().toString());
        return message(TOPIC_PAYMENT_RECEIVED, "PaymentReceived", bill.appointmentId(), "payment", body);
    }

    /** An attempt was declined; the customer can retry on the same payment. */
    static OutboxMessage paymentFailed(Bill bill, Payment payment) {
        Map<String, Object> body = paymentBody(bill, payment);
        body.put("failure_code", payment.failureCode());
        body.put("failure_message", payment.failureMessage());
        return message(TOPIC_PAYMENT_FAILED, "PaymentFailed", bill.appointmentId(), "payment", body);
    }

    private static Map<String, Object> paymentBody(Bill bill, Payment payment) {
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("id", payment.id());
        body.put("bill_id", bill.id());
        body.put("appointment_id", bill.appointmentId());
        body.put("customer_id", bill.customerId());
        body.put("type", payment.type().name());
        body.put("amount_minor", payment.amount().amountMinor());
        body.put("currency", payment.amount().currency());
        return body;
    }

    private static OutboxMessage message(String topic, String eventType, long appointmentId, String entity,
            Map<String, Object> body) {
        Map<String, Object> envelope = new LinkedHashMap<>();
        envelope.put("event_id", UUID.randomUUID().toString());
        envelope.put("event_type", eventType);
        envelope.put("occurred_at", Instant.now().toString());
        envelope.put(entity, body);
        return OutboxMessage.of(topic, Long.toString(appointmentId), JSON.writeValueAsString(envelope));
    }
}
