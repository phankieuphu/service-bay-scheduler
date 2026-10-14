package com.servicebay.billing;

import static org.assertj.core.api.Assertions.assertThat;

import com.nimbusds.jose.jwk.source.ImmutableSecret;
import com.nimbusds.jose.proc.SecurityContext;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import javax.crypto.spec.SecretKeySpec;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.resttestclient.TestRestTemplate;
import org.springframework.boot.resttestclient.autoconfigure.AutoConfigureTestRestTemplate;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.SpringBootTest.WebEnvironment;
import org.springframework.context.annotation.Import;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.http.ResponseEntity;
import org.springframework.security.oauth2.jose.jws.MacAlgorithm;
import org.springframework.security.oauth2.jwt.JwsHeader;
import org.springframework.security.oauth2.jwt.JwtClaimsSet;
import org.springframework.security.oauth2.jwt.JwtEncoderParameters;
import org.springframework.security.oauth2.jwt.NimbusJwtEncoder;

/** #90: which routes need identity-service's token, and that a bad one is refused. */
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@AutoConfigureTestRestTemplate
@Import(TestcontainersConfiguration.class)
class SecurityIT {

    private static final String SECRET = "test-jwt-secret-0123456789-0123456789"; // src/test/resources/config

    @Autowired
    TestRestTemplate http;

    private static String token(String secret, String issuer, String role, Instant expiresAt) {
        var encoder = new NimbusJwtEncoder(new ImmutableSecret<SecurityContext>(
                new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256")));
        JwtClaimsSet claims = JwtClaimsSet.builder().issuer(issuer).subject("user-1").claim("role", role)
                .issuedAt(expiresAt.minusSeconds(900)).expiresAt(expiresAt).build();
        return encoder.encode(JwtEncoderParameters.from(JwsHeader.with(MacAlgorithm.HS256).build(), claims)).getTokenValue();
    }

    private ResponseEntity<String> get(String path, String bearer) {
        HttpHeaders headers = new HttpHeaders();
        if (bearer != null) {
            headers.setBearerAuth(bearer);
        }
        return http.exchange(path, HttpMethod.GET, new HttpEntity<>(headers), String.class);
    }

    @Test
    void apiNeedsAValidIdentityToken() {
        Instant later = Instant.now().plusSeconds(600);
        assertThat(get("/api/v1/bills/1", null).getStatusCode().value()).isEqualTo(401);
        assertThat(get("/api/v1/bills/1", token("x".repeat(40), "identity-service", "ADMIN", later))
                .getStatusCode().value()).isEqualTo(401);
        assertThat(get("/api/v1/bills/1", token(SECRET, "someone-else", "ADMIN", later))
                .getStatusCode().value()).isEqualTo(401);
        assertThat(get("/api/v1/bills/1", token(SECRET, "identity-service", "ADMIN", Instant.now().minusSeconds(120)))
                .getStatusCode().value()).isEqualTo(401);
    }

    @Test
    void aValidStaffTokenReachesTheApi() {
        ResponseEntity<String> response = get("/api/v1/bills/987654",
                token(SECRET, "identity-service", "ADMIN", Instant.now().plusSeconds(600)));

        assertThat(response.getStatusCode().value()).isEqualTo(404);
        assertThat(response.getBody()).contains("\"error\":\"bill not found\"");
    }

    @Test
    void webhookIsOpenButNeedsItsSecret() {
        ResponseEntity<String> response = http.postForEntity("/webhooks/stripe", "{}", String.class);

        // Reached the controller (not 401); no STRIPE_WEBHOOK_SECRET in tests, so 503.
        assertThat(response.getStatusCode().value()).isEqualTo(503);
    }
}
