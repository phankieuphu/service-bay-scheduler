package com.payment.payment_service.gateway;

import java.math.BigDecimal;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import com.payment.payment_service.config.StripeProperties;
import com.payment.payment_service.exceptions.GatewayException;
import com.payment.payment_service.exceptions.GatewayNotConfiguredException;
import com.payment.payment_service.exceptions.InvalidWebhookException;
import com.payment.payment_service.ports.PaymentGateway;
import com.stripe.StripeClient;
import com.stripe.exception.CardException;
import com.stripe.exception.EventDataObjectDeserializationException;
import com.stripe.exception.InvalidRequestException;
import com.stripe.exception.SignatureVerificationException;
import com.stripe.exception.StripeException;
import com.stripe.model.Event;
import com.stripe.model.PaymentIntent;
import com.stripe.model.StripeObject;
import com.stripe.net.RequestOptions;
import com.stripe.net.Webhook;
import com.stripe.param.PaymentIntentConfirmParams;
import com.stripe.param.PaymentIntentCreateParams;

@Component
public class StripePaymentGateway implements PaymentGateway {

  private static final Logger log = LoggerFactory.getLogger(StripePaymentGateway.class);

  // Stripe error code for an operation the intent's current status doesn't allow,
  // e.g. canceling an intent that already succeeded.
  private static final String UNEXPECTED_STATE = "payment_intent_unexpected_state";

  static final String METADATA_PAYMENT_ID = "payment_id";
  static final String METADATA_BILL_ID = "bill_id";

  private final StripeProperties properties;
  private final StripeClient client;

  public StripePaymentGateway(StripeProperties properties) {
    this.properties = properties;
    this.client = properties.hasSecretKey() ? buildClient(properties) : null;
    if (client == null) {
      log.warn("STRIPE_SECRET_KEY is not set: card payment endpoints will answer 503");
    }
    if (!properties.hasWebhookSecret()) {
      log.warn("STRIPE_WEBHOOK_SECRET is not set: the Stripe webhook endpoint will answer 503");
    }
  }

  @Override
  public GatewayIntent createIntent(long paymentId, long billId, BigDecimal amount, String currency) {
    PaymentIntentCreateParams params = PaymentIntentCreateParams.builder()
        .setAmount(Money.toMinorUnits(amount, currency))
        .setCurrency(Money.normalizeCurrency(currency))
        .setAutomaticPaymentMethods(PaymentIntentCreateParams.AutomaticPaymentMethods.builder()
            .setEnabled(true)
            .build())
        .putMetadata(METADATA_PAYMENT_ID, Long.toString(paymentId))
        .putMetadata(METADATA_BILL_ID, Long.toString(billId))
        .build();
    // Same key for every retry of the same payment: Stripe returns the original
    // intent instead of creating a second one.
    RequestOptions options = RequestOptions.builder()
        .setIdempotencyKey("service-bay-payment-" + paymentId)
        .build();
    try {
      return toIntent(client().v1().paymentIntents().create(params, options));
    } catch (StripeException e) {
      throw translate("create payment intent", e);
    }
  }

  @Override
  public GatewayIntent retrieveIntent(String intentId) {
    try {
      return toIntent(client().v1().paymentIntents().retrieve(intentId));
    } catch (StripeException e) {
      throw translate("retrieve payment intent " + intentId, e);
    }
  }

  @Override
  public GatewayIntent cancelIntent(String intentId) {
    try {
      return toIntent(client().v1().paymentIntents().cancel(intentId));
    } catch (StripeException e) {
      throw translate("cancel payment intent " + intentId, e);
    }
  }

  @Override
  public GatewayIntent confirmIntent(String intentId, String paymentMethod, String returnUrl) {
    PaymentIntentConfirmParams params = PaymentIntentConfirmParams.builder()
        .setPaymentMethod(paymentMethod)
        // Required because automatic payment methods may redirect; a test card never does.
        .setReturnUrl(returnUrl)
        .build();
    try {
      return toIntent(client().v1().paymentIntents().confirm(intentId, params));
    } catch (CardException e) {
      // Declined: the intent stays open with last_payment_error set. Read it back so
      // the caller sees the same state a payment_intent.payment_failed webhook carries.
      log.info("Stripe declined confirmation of {} (decline code {})", intentId, e.getDeclineCode());
      return retrieveIntent(intentId);
    } catch (StripeException e) {
      throw translate("confirm payment intent " + intentId, e);
    }
  }

  @Override
  public boolean isTestMode() {
    return properties.hasSecretKey() && properties.secretKey().startsWith("sk_test_");
  }

  @Override
  public WebhookEvent parseWebhook(String payload, String signatureHeader) {
    if (!properties.hasWebhookSecret()) {
      throw new GatewayNotConfiguredException("Stripe webhook secret is not configured");
    }
    Event event;
    try {
      event = Webhook.constructEvent(payload, signatureHeader, properties.webhookSecret());
    } catch (SignatureVerificationException e) {
      throw new InvalidWebhookException("invalid Stripe signature", e);
    } catch (RuntimeException e) {
      // Missing header or a body that isn't a Stripe event.
      throw new InvalidWebhookException("malformed Stripe webhook", e);
    }

    if (!(dataObject(event) instanceof PaymentIntent intent)) {
      return new WebhookEvent(event.getId(), event.getType(), null, null, null, null);
    }
    return new WebhookEvent(event.getId(), event.getType(), intent.getId(),
        parseLong(intent.getMetadata() == null ? null : intent.getMetadata().get(METADATA_PAYMENT_ID)),
        intent.getAmountReceived(), failureReason(intent));
  }

  private static StripeClient buildClient(StripeProperties properties) {
    StripeClient.StripeClientBuilder builder = StripeClient.builder()
        .setApiKey(properties.secretKey())
        // Retries network errors and 409/5xx with backoff. Safe: create carries an
        // explicit idempotency key and the SDK adds one to the other POSTs.
        .setMaxNetworkRetries(2);
    if (properties.hasApiBase()) {
      builder.setApiBase(properties.apiBase());
    }
    return builder.build();
  }

  private StripeClient client() {
    if (client == null) {
      throw new GatewayNotConfiguredException("Stripe secret key is not configured");
    }
    return client;
  }

  // getObject() is empty when the event's API version differs from the one this
  // SDK is pinned to (e.g. the webhook endpoint uses an older version); the
  // PaymentIntent fields read here are stable across versions, so fall back.
  private static StripeObject dataObject(Event event) {
    var deserializer = event.getDataObjectDeserializer();
    return deserializer.getObject().orElseGet(() -> {
      try {
        return deserializer.deserializeUnsafe();
      } catch (EventDataObjectDeserializationException e) {
        log.warn("could not deserialize Stripe event {} ({})", event.getId(), event.getType(), e);
        return null;
      }
    });
  }

  private static String failureReason(PaymentIntent intent) {
    if (intent.getLastPaymentError() != null) {
      return intent.getLastPaymentError().getMessage();
    }
    if (intent.getCancellationReason() != null) {
      return "canceled: " + intent.getCancellationReason();
    }
    return null;
  }

  private static Long parseLong(String value) {
    if (value == null) {
      return null;
    }
    try {
      return Long.valueOf(value);
    } catch (NumberFormatException e) {
      return null;
    }
  }

  private static GatewayIntent toIntent(PaymentIntent intent) {
    return new GatewayIntent(intent.getId(), intent.getClientSecret(), intent.getStatus(),
        intent.getAmountReceived(), failureReason(intent));
  }

  private static GatewayException translate(String action, StripeException e) {
    boolean conflict = e instanceof InvalidRequestException && UNEXPECTED_STATE.equals(e.getCode());
    // Log Stripe's request id: it is what Stripe support asks for.
    log.warn("Stripe failed to {} (request {}, code {}): {}", action, e.getRequestId(), e.getCode(), e.getMessage());
    return new GatewayException("payment gateway failed to " + action + ": " + e.getMessage(), e, conflict);
  }
}
