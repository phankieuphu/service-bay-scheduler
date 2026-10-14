package com.servicebay.billing.adapters.stripe;

import org.springframework.boot.context.properties.ConfigurationProperties;

/**
 * Stripe keys (#83). Use test-mode keys outside production. With no secret
 * key the service still starts; card endpoints answer 503.
 *
 * @param apiBase empty for api.stripe.com; set to point the SDK at stripe-mock
 * @param returnUrl where a server-side test confirmation says the customer returns to (test mode only)
 */
@ConfigurationProperties(prefix = "stripe")
public record StripeProperties(String secretKey, String webhookSecret, String publishableKey, String apiBase,
        String returnUrl) {

    boolean hasSecretKey() {
        return secretKey != null && !secretKey.isBlank();
    }

    boolean hasWebhookSecret() {
        return webhookSecret != null && !webhookSecret.isBlank();
    }
}
