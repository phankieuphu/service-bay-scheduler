package com.servicebay.billing.domain.payment;

import com.servicebay.billing.domain.money.Money;
import java.util.Objects;

public record Refund(
        Long id,
        long paymentId,
        Money amount,
        String stripeRefundId,
        RefundStatus status,
        String reason,
        long version) {

    public Refund {
        Objects.requireNonNull(amount, "amount");
        Objects.requireNonNull(status, "status");
        if (!amount.isPositive()) {
            throw new IllegalArgumentException("refund amount must be positive");
        }
    }
}
