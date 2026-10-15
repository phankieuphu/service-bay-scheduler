package com.servicebay.billing.adapters.security;

import java.time.Duration;
import java.time.Instant;
import org.springframework.security.oauth2.jose.jws.MacAlgorithm;
import org.springframework.security.oauth2.jwt.JwsHeader;
import org.springframework.security.oauth2.jwt.JwtClaimsSet;
import org.springframework.security.oauth2.jwt.JwtEncoder;
import org.springframework.security.oauth2.jwt.JwtEncoderParameters;
import org.springframework.stereotype.Component;

/**
 * Short-lived tokens for billing's own calls, where there's no user token to
 * forward: a Kafka handler looking up a warranty. Signed with the shared
 * secret like identity-service's, with subject {@code service:billing-service}
 * and role {@code SERVICE}. The Go services let any signed-in principal read,
 * and none of their write routes accept SERVICE, so it can only read.
 */
@Component
public class ServiceTokenIssuer {

    private static final Duration TTL = Duration.ofMinutes(5);

    private final JwtEncoder encoder;
    private final JwtProperties properties;

    ServiceTokenIssuer(JwtEncoder encoder, JwtProperties properties) {
        this.encoder = encoder;
        this.properties = properties;
    }

    public String token() {
        Instant now = Instant.now();
        JwtClaimsSet claims = JwtClaimsSet.builder()
                .issuer(properties.issuer())
                .subject("service:billing-service")
                .claim("role", "SERVICE")
                .issuedAt(now)
                .expiresAt(now.plus(TTL))
                .build();
        return encoder.encode(JwtEncoderParameters.from(JwsHeader.with(MacAlgorithm.HS256).build(), claims))
                .getTokenValue();
    }
}
