package com.servicebay.billing;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * billing-service: deposits, bills and payments, with Stripe as the payment
 * provider. Packages mirror the Go services' hexagonal layout:
 *
 * <ul>
 *   <li>{@code domain}: entities, ports (interfaces) and services. No Spring, JPA or Stripe imports.
 *   <li>{@code adapters.http}, {@code adapters.persistence}, {@code adapters.kafka},
 *       {@code adapters.stripe}: implementations of the ports.
 *   <li>{@code application}: wiring (Spring configuration).
 * </ul>
 */
@SpringBootApplication
public class BillingApplication {

    public static void main(String[] args) {
        SpringApplication.run(BillingApplication.class, args);
    }
}
