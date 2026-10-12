package com.servicebay.billing.domain.ports;

import java.util.Optional;

/** Maps a Service Bay customer to their Stripe Customer, created lazily. */
public interface StripeCustomerRepository {

    Optional<String> findStripeCustomerId(long customerId);

    /** Throws {@link ConflictException} if the customer already has one (a concurrent create won). */
    void save(long customerId, String stripeCustomerId);
}
