package com.servicebay.billing.adapters.stripe;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.servicebay.billing.domain.ports.PaymentProvider.ProviderEvent;
import com.servicebay.billing.domain.ports.PaymentProviderException;
import com.servicebay.billing.domain.ports.PaymentProviderException.Kind;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.HexFormat;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import org.junit.jupiter.api.Test;

/** The real stripe-java signature check, with a locally signed payload: no network. */
class StripePaymentProviderTest {

    private static final String SECRET = "whsec_test_secret";
    private static final String PAYLOAD = """
            {"id":"evt_123","object":"event","type":"payment_intent.succeeded","api_version":"%s",
             "data":{"object":{"id":"pi_123","object":"payment_intent","amount":2500,"amount_received":2500,
             "currency":"usd","status":"succeeded","metadata":{"payment_id":"7"}}}}
            """.formatted(com.stripe.Stripe.API_VERSION);

    private final StripePaymentProvider provider = new StripePaymentProvider(
            new StripeProperties(null, SECRET, null, null, "http://localhost/return"));

    private static String sign(String payload, String secret) throws Exception {
        long t = Instant.now().getEpochSecond();
        Mac mac = Mac.getInstance("HmacSHA256");
        mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
        return "t=" + t + ",v1=" + HexFormat.of().formatHex(mac.doFinal((t + "." + payload).getBytes(StandardCharsets.UTF_8)));
    }

    @Test
    void parsesASignedEvent() throws Exception {
        ProviderEvent event = provider.parseWebhook(PAYLOAD, sign(PAYLOAD, SECRET));

        assertThat(event).isEqualTo(new ProviderEvent("evt_123", "payment_intent.succeeded", "pi_123"));
    }

    @Test
    void rejectsABadSignatureATamperedBodyAndAMissingHeader() throws Exception {
        String foreign = sign(PAYLOAD, "whsec_someone_else");
        String good = sign(PAYLOAD, SECRET);

        assertThatThrownBy(() -> provider.parseWebhook(PAYLOAD, foreign)).satisfies(kind(Kind.INVALID_WEBHOOK));
        assertThatThrownBy(() -> provider.parseWebhook(PAYLOAD.replace("2500", "1"), good)).satisfies(kind(Kind.INVALID_WEBHOOK));
        assertThatThrownBy(() -> provider.parseWebhook(PAYLOAD, null)).satisfies(kind(Kind.INVALID_WEBHOOK));
    }

    @Test
    void withoutKeysPaymentsAre503AndTestConfirmIsRefused() {
        assertThatThrownBy(() -> provider.retrieveIntent("pi_1")).satisfies(kind(Kind.NOT_CONFIGURED));
        assertThat(provider.isTestMode()).isFalse();
        assertThatThrownBy(() -> provider.confirmTestIntent("pi_1", "pm_card_visa")).satisfies(kind(Kind.TEST_MODE_ONLY));

        StripePaymentProvider live = new StripePaymentProvider(new StripeProperties("sk_live_x", null, null, null, null));
        assertThat(live.isTestMode()).isFalse();
        assertThat(new StripePaymentProvider(new StripeProperties("sk_test_x", null, null, null, null)).isTestMode()).isTrue();
    }

    private static java.util.function.Consumer<Throwable> kind(Kind expected) {
        return e -> assertThat(((PaymentProviderException) e).kind()).isEqualTo(expected);
    }
}
