package com.payment.payment_service.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

// apiBase is empty in real use; set it to point the SDK at stripe-mock
// (https://github.com/stripe/stripe-mock) for local end-to-end runs.
@ConfigurationProperties(prefix = "stripe")
public record StripeProperties(String secretKey, String webhookSecret, String apiBase) {

  public boolean hasSecretKey() {
    return secretKey != null && !secretKey.isBlank();
  }

  public boolean hasApiBase() {
    return apiBase != null && !apiBase.isBlank();
  }

  public boolean hasWebhookSecret() {
    return webhookSecret != null && !webhookSecret.isBlank();
  }
}
