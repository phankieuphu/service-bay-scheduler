package com.servicebay.billing.adapters.stripe;

import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.ports.PaymentProvider;
import com.servicebay.billing.domain.ports.PaymentProviderException;
import com.servicebay.billing.domain.ports.PaymentProviderException.Kind;
import com.stripe.StripeClient;
import com.stripe.exception.CardException;
import com.stripe.exception.InvalidRequestException;
import com.stripe.exception.SignatureVerificationException;
import com.stripe.exception.StripeException;
import com.stripe.model.Charge;
import com.stripe.model.Event;
import com.stripe.model.PaymentIntent;
import com.stripe.model.StripeObject;
import com.stripe.net.RequestOptions;
import com.stripe.net.Webhook;
import com.stripe.param.CustomerCreateParams;
import com.stripe.param.PaymentIntentConfirmParams;
import com.stripe.param.PaymentIntentCreateParams;
import com.stripe.param.PaymentIntentRetrieveParams;
import java.util.Locale;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

/** The only class that imports stripe-java (#83). */
@Component
class StripePaymentProvider implements PaymentProvider {

    private static final Logger log = LoggerFactory.getLogger(StripePaymentProvider.class);

    /** Stripe's code for an operation the intent's status doesn't allow. */
    private static final String UNEXPECTED_STATE = "payment_intent_unexpected_state";

    /** The metadata key the use cases put our payment id under. */
    static final String METADATA_PAYMENT_ID = "payment_id";

    private final StripeProperties properties;
    private final StripeClient client;

    StripePaymentProvider(StripeProperties properties) {
        this.properties = properties;
        this.client = properties.hasSecretKey() ? buildClient(properties) : null;
        if (client == null) {
            log.warn("STRIPE_SECRET_KEY is not set: payment endpoints will answer 503");
        }
        if (!properties.hasWebhookSecret()) {
            log.warn("STRIPE_WEBHOOK_SECRET is not set: POST /webhooks/stripe will answer 503");
        }
    }

    private static StripeClient buildClient(StripeProperties properties) {
        StripeClient.StripeClientBuilder builder = StripeClient.builder()
                .setApiKey(properties.secretKey())
                // Retries network errors and 409/5xx with backoff. Safe: creates
                // carry our idempotency key, and the SDK adds one to other POSTs.
                .setMaxNetworkRetries(2);
        if (properties.apiBase() != null && !properties.apiBase().isBlank()) {
            builder.setApiBase(properties.apiBase());
        }
        return builder.build();
    }

    @Override
    public String createCustomer(long customerId, String idempotencyKey) {
        CustomerCreateParams params = CustomerCreateParams.builder()
                .putMetadata("customer_id", Long.toString(customerId))
                .build();
        try {
            return client().v1().customers().create(params, options(idempotencyKey)).getId();
        } catch (StripeException e) {
            throw translate("create a customer for " + customerId, e);
        }
    }

    @Override
    public ProviderIntent createIntent(Money amount, String providerCustomerId, Map<String, String> metadata,
            String idempotencyKey) {
        PaymentIntentCreateParams.Builder params = PaymentIntentCreateParams.builder()
                .setAmount(amount.amountMinor())
                .setCurrency(amount.currency().toLowerCase(Locale.ROOT))
                .setCustomer(providerCustomerId)
                .setAutomaticPaymentMethods(PaymentIntentCreateParams.AutomaticPaymentMethods.builder()
                        .setEnabled(true)
                        .build());
        metadata.forEach(params::putMetadata);
        try {
            return toIntent(client().v1().paymentIntents().create(params.build(), options(idempotencyKey)));
        } catch (StripeException e) {
            throw translate("create a payment intent", e);
        }
    }

    @Override
    public ProviderIntent retrieveIntent(String intentId) {
        // latest_charge expanded: the card brand and last4 live on the charge.
        PaymentIntentRetrieveParams params = PaymentIntentRetrieveParams.builder().addExpand("latest_charge").build();
        try {
            return toIntent(client().v1().paymentIntents().retrieve(intentId, params));
        } catch (StripeException e) {
            throw translate("retrieve payment intent " + intentId, e);
        }
    }

    @Override
    public ProviderIntent cancelIntent(String intentId) {
        try {
            return toIntent(client().v1().paymentIntents().cancel(intentId));
        } catch (StripeException e) {
            throw translate("cancel payment intent " + intentId, e);
        }
    }

    @Override
    public ProviderIntent confirmTestIntent(String intentId, String paymentMethod) {
        if (!isTestMode()) {
            throw new PaymentProviderException(Kind.TEST_MODE_ONLY, "test confirmation needs a Stripe test-mode key (sk_test_...)");
        }
        PaymentIntentConfirmParams params = PaymentIntentConfirmParams.builder()
                .setPaymentMethod(paymentMethod)
                // Required with automatic payment methods; a test card never redirects.
                .setReturnUrl(properties.returnUrl())
                .build();
        try {
            client().v1().paymentIntents().confirm(intentId, params);
        } catch (CardException e) {
            // Declined: the intent stays open with last_payment_error set, as a
            // payment_intent.payment_failed webhook would report it.
            log.info("Stripe declined test confirmation of {} ({})", intentId, e.getDeclineCode());
        } catch (StripeException e) {
            throw translate("confirm payment intent " + intentId, e);
        }
        return retrieveIntent(intentId);
    }

    @Override
    public boolean isTestMode() {
        return properties.hasSecretKey() && properties.secretKey().startsWith("sk_test_");
    }

    @Override
    public ProviderEvent parseWebhook(String payload, String signatureHeader) {
        if (!properties.hasWebhookSecret()) {
            throw new PaymentProviderException(Kind.NOT_CONFIGURED, "Stripe webhook secret is not configured");
        }
        Event event;
        try {
            event = Webhook.constructEvent(payload, signatureHeader, properties.webhookSecret());
        } catch (SignatureVerificationException e) {
            throw new PaymentProviderException(Kind.INVALID_WEBHOOK, "invalid Stripe signature", e);
        } catch (RuntimeException e) {
            // Missing header, or a body that isn't a Stripe event.
            throw new PaymentProviderException(Kind.INVALID_WEBHOOK, "malformed Stripe webhook", e);
        }
        String intentId = dataObject(event) instanceof PaymentIntent intent ? intent.getId() : null;
        return new ProviderEvent(event.getId(), event.getType(), intentId);
    }

    private StripeClient client() {
        if (client == null) {
            throw new PaymentProviderException(Kind.NOT_CONFIGURED, "Stripe secret key is not configured");
        }
        return client;
    }

    private static RequestOptions options(String idempotencyKey) {
        return RequestOptions.builder().setIdempotencyKey(idempotencyKey).build();
    }

    // getObject() is empty when the event's API version differs from the
    // SDK's pinned one; only the intent id is read here, which is stable.
    private static StripeObject dataObject(Event event) {
        var deserializer = event.getDataObjectDeserializer();
        return deserializer.getObject().orElseGet(() -> {
            try {
                return deserializer.deserializeUnsafe();
            } catch (Exception e) {
                log.warn("could not deserialize Stripe event {} ({})", event.getId(), event.getType(), e);
                return null;
            }
        });
    }

    private static ProviderIntent toIntent(PaymentIntent intent) {
        String code = null;
        String message = null;
        if (intent.getLastPaymentError() != null) {
            code = intent.getLastPaymentError().getDeclineCode() != null
                    ? intent.getLastPaymentError().getDeclineCode()
                    : intent.getLastPaymentError().getCode();
            message = intent.getLastPaymentError().getMessage();
        } else if (intent.getCancellationReason() != null) {
            code = "canceled";
            message = "canceled: " + intent.getCancellationReason();
        }
        String brand = null;
        String last4 = null;
        Charge charge = intent.getLatestChargeObject();
        if (charge != null && charge.getPaymentMethodDetails() != null && charge.getPaymentMethodDetails().getCard() != null) {
            brand = charge.getPaymentMethodDetails().getCard().getBrand();
            last4 = charge.getPaymentMethodDetails().getCard().getLast4();
        }
        return new ProviderIntent(intent.getId(), intent.getClientSecret(), intent.getStatus(),
                intent.getAmount() == null ? 0 : intent.getAmount(),
                intent.getAmountReceived() == null ? 0 : intent.getAmountReceived(),
                code, message, brand, last4, paymentId(intent));
    }

    private static Long paymentId(PaymentIntent intent) {
        String value = intent.getMetadata() == null ? null : intent.getMetadata().get(METADATA_PAYMENT_ID);
        try {
            return value == null ? null : Long.valueOf(value);
        } catch (NumberFormatException e) {
            return null;
        }
    }

    private static PaymentProviderException translate(String action, StripeException e) {
        Kind kind = e instanceof InvalidRequestException && UNEXPECTED_STATE.equals(e.getCode())
                ? Kind.CONFLICT
                : Kind.UNAVAILABLE;
        // Stripe's request id is what Stripe support asks for.
        log.warn("Stripe failed to {} (request {}, code {}): {}", action, e.getRequestId(), e.getCode(), e.getMessage());
        return new PaymentProviderException(kind, "payment provider failed to " + action + ": " + e.getMessage(), e);
    }
}
