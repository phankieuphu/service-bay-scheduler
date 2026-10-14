package com.servicebay.billing.domain.deposit;

import com.servicebay.billing.domain.money.Money;
import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.Locale;
import java.util.Objects;

/**
 * How much a deposit at booking is (business-requirement §10 #5 is still
 * open, so it's configuration, #86):
 * <ul>
 *   <li>{@code FIXED}: {@code value} in major units of the booking's currency, capped at the estimate</li>
 *   <li>{@code PERCENT}: {@code value} percent of the booking's estimate, rounded up to a whole minor unit</li>
 *   <li>{@code FULL}: the whole estimate</li>
 * </ul>
 */
public record DepositPolicy(Kind kind, BigDecimal value) {

    public enum Kind { FIXED, PERCENT, FULL }

    public DepositPolicy {
        Objects.requireNonNull(kind, "kind");
        value = value == null ? BigDecimal.ZERO : value;
        if (kind == Kind.PERCENT && (value.signum() <= 0 || value.compareTo(BigDecimal.valueOf(100)) > 0)) {
            throw new IllegalArgumentException("a percent deposit must be in (0, 100], got " + value);
        }
        if (kind == Kind.FIXED && value.signum() <= 0) {
            throw new IllegalArgumentException("a fixed deposit must be positive, got " + value);
        }
    }

    public static DepositPolicy parse(String kind, BigDecimal value) {
        return new DepositPolicy(Kind.valueOf(kind.trim().toUpperCase(Locale.ROOT)), value);
    }

    /**
     * The deposit for a booking with this estimate. Zero means no deposit is
     * due (e.g. a free warranty inspection); the caller should refuse it.
     */
    public Money amountFor(Money estimate) {
        return switch (kind) {
            case FULL -> estimate;
            case FIXED -> min(Money.ofMajor(value, estimate.currency()), estimate);
            case PERCENT -> Money.of(BigDecimal.valueOf(estimate.amountMinor())
                    .multiply(value)
                    .divide(BigDecimal.valueOf(100), 0, RoundingMode.CEILING)
                    .longValueExact(), estimate.currency());
        };
    }

    private static Money min(Money a, Money b) {
        return a.compareTo(b) <= 0 ? a : b;
    }
}
