package com.servicebay.billing.adapters.security;

import org.springframework.boot.context.properties.ConfigurationProperties;

/** Must match identity-service's JWT_SECRET (at least 32 bytes) and JWT_ISSUER. */
@ConfigurationProperties(prefix = "jwt")
public record JwtProperties(String secret, String issuer) {
}
