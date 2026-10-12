package com.servicebay.billing.domain.ports;

import com.servicebay.billing.domain.payment.Refund;
import java.util.List;
import java.util.Optional;

public interface RefundRepository {

    Refund save(Refund refund);

    Optional<Refund> findByStripeRefundId(String stripeRefundId);

    List<Refund> findByPayment(long paymentId);

    /** Every refund of every payment on the bill. */
    List<Refund> findByBill(long billId);
}
