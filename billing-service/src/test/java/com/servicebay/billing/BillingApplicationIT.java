package com.servicebay.billing;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.SpringBootTest.WebEnvironment;
import org.springframework.boot.resttestclient.TestRestTemplate;
import org.springframework.boot.resttestclient.autoconfigure.AutoConfigureTestRestTemplate;
import org.springframework.context.annotation.Import;

/** Starts the whole service against a real Postgres (Testcontainers). */
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@AutoConfigureTestRestTemplate
@Import(TestcontainersConfiguration.class)
class BillingApplicationIT {

    @Autowired
    TestRestTemplate http;

    @Test
    void readinessIsUpOncePostgresIsReachable() {
        var response = http.getForEntity("/actuator/health/readiness", String.class);
        assertThat(response.getStatusCode().value()).isEqualTo(200);
        assertThat(response.getBody()).contains("\"status\":\"UP\"");
    }

    @Test
    void livenessAndPrometheusAreExposed() {
        assertThat(http.getForEntity("/actuator/health/liveness", String.class).getStatusCode().value()).isEqualTo(200);
        var metrics = http.getForEntity("/actuator/prometheus", String.class);
        assertThat(metrics.getStatusCode().value()).isEqualTo(200);
        assertThat(metrics.getBody()).contains("application=\"billing-service\"");
    }
}
