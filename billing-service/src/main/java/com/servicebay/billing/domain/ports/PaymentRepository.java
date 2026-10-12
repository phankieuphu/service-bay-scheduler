package com.servicebay.billing.domain.ports;

import com.servicebay.billing.domain.payment.Payment;
import java.util.List;
import java.util.Optional;

public interface PaymentRepository {

    /** Throws {@link ConflictException} on a duplicate idempotency key or PaymentIntent id. */
    Payment save(Payment payment);

    Optional<Payment> findById(long id);

    Optional<Payment> findByIdempotencyKey(String idempotencyKey);

    Optional<Payment> findByStripePaymentIntentId(String paymentIntentId);

    List<Payment> findByBill(long billId);
}
