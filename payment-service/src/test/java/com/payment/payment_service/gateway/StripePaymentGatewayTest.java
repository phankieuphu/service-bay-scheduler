package com.payment.payment_service.gateway;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.HexFormat;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;

import org.junit.jupiter.api.Test;

import com.payment.payment_service.config.StripeProperties;
import com.payment.payment_service.exceptions.GatewayNotConfiguredException;
import com.payment.payment_service.exceptions.InvalidWebhookException;
import com.payment.payment_service.ports.PaymentGateway.WebhookEvent;

// Exercises the real stripe-java signature check with a locally signed payload;
// no network access.
class StripePaymentGatewayTest {

  private static final String SECRET = "whsec_test_secret";

  private static final String PAYLOAD = """
      {
        "id": "evt_123",
        "object": "event",
        "type": "payment_intent.succeeded",
        "api_version": "%s",
        "data": {
          "object": {
            "id": "pi_123",
            "object": "payment_intent",
            "amount": 2500,
            "amount_received": 2500,
            "currency": "usd",
            "status": "succeeded",
            "metadata": { "payment_id": "7", "bill_id": "1" }
          }
        }
      }
      """.formatted(com.stripe.Stripe.API_VERSION);

  private final StripePaymentGateway gateway = new StripePaymentGateway(new StripeProperties(null, SECRET, null));

  private static String sign(String payload, String secret) throws Exception {
    long timestamp = Instant.now().getEpochSecond();
    Mac mac = Mac.getInstance("HmacSHA256");
    mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
    String signature = HexFormat.of().formatHex(
        mac.doFinal((timestamp + "." + payload).getBytes(StandardCharsets.UTF_8)));
    return "t=" + timestamp + ",v1=" + signature;
  }

  @Test
  void parsesSignedPaymentIntentEvent() throws Exception {
    WebhookEvent event = gateway.parseWebhook(PAYLOAD, sign(PAYLOAD, SECRET));

    assertEquals("evt_123", event.id());
    assertEquals("payment_intent.succeeded", event.type());
    assertEquals("pi_123", event.intentId());
    assertEquals(7L, event.paymentId());
    assertEquals(2500L, event.amountReceived());
    assertNull(event.failureReason());
  }

  @Test
  void rejectsWrongSignature() throws Exception {
    String header = sign(PAYLOAD, "whsec_someone_else");

    assertThrows(InvalidWebhookException.class, () -> gateway.parseWebhook(PAYLOAD, header));
  }

  @Test
  void rejectsTamperedBody() throws Exception {
    String header = sign(PAYLOAD, SECRET);

    assertThrows(InvalidWebhookException.class,
        () -> gateway.parseWebhook(PAYLOAD.replace("2500", "1"), header));
  }

  @Test
  void rejectsMissingSignatureHeader() {
    assertThrows(InvalidWebhookException.class, () -> gateway.parseWebhook(PAYLOAD, null));
  }

  @Test
  void unconfiguredSecretKeyFailsCardCallsWith503() {
    assertThrows(GatewayNotConfiguredException.class, () -> gateway.retrieveIntent("pi_123"));
  }
}
