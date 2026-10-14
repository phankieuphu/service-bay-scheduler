package com.servicebay.billing;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;

/**
 * billing-service: deposits, bills and payments, with Stripe as the payment
 * provider. Packages mirror the Go services' hexagonal layout:
 *
 * <ul>
 *   <li>{@code domain}: entities, ports (interfaces) and services. No Spring, JPA or Stripe imports.
 *   <li>{@code adapters.http}, {@code adapters.persistence}, {@code adapters.kafka},
 *       {@code adapters.stripe}, {@code adapters.security}, {@code adapters.upstream}: implementations of the
 *       ports, plus the HTTP API and the Kafka consumers.
 *   <li>{@code application}: use cases (deposit, pay, webhooks, pricing, reads), each drawing its own
 *       transaction boundaries, and Spring wiring.
 * </ul>
 */
@SpringBootApplication
@ConfigurationPropertiesScan
public class BillingApplication {

    public static void main(String[] args) {
        SpringApplication.run(BillingApplication.class, args);
    }
}
