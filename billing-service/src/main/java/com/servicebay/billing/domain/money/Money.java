package com.servicebay.billing.domain.money;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.Currency;
import java.util.Objects;
import java.util.Set;

/**
 * An amount of money: integer minor units plus an ISO 4217 currency. Never a
 * {@code double}. Minor units follow Stripe's convention, since that's where
 * amounts are sent: most currencies have 2 decimals (1000 USD minor = $10.00),
 * zero-decimal currencies such as VND and JPY have none (1000 VND minor =
 * 1000 ₫), and a few have 3.
 */
public record Money(long amountMinor, String currency) implements Comparable<Money> {

    /** Stripe's zero-decimal currencies: the amount is already in whole units. */
    private static final Set<String> ZERO_DECIMAL = Set.of(
            "BIF", "CLP", "DJF", "GNF", "JPY", "KMF", "KRW", "MGA",
            "PYG", "RWF", "UGX", "VND", "VUV", "XAF", "XOF", "XPF");

    /** Stripe's three-decimal currencies. */
    private static final Set<String> THREE_DECIMAL = Set.of("BHD", "JOD", "KWD", "OMR", "TND");

    public Money {
        Objects.requireNonNull(currency, "currency");
        currency = currency.toUpperCase();
        try {
            Currency.getInstance(currency);
        } catch (IllegalArgumentException e) {
            throw new IllegalArgumentException("unknown ISO 4217 currency: " + currency, e);
        }
    }

    public static Money of(long amountMinor, String currency) {
        return new Money(amountMinor, currency);
    }

    public static Money zero(String currency) {
        return new Money(0, currency);
    }

    /**
     * Converts a major-unit amount ("12.34" USD, "150000" VND). It must fit the
     * currency's decimals exactly; an amount with more precision is rejected
     * rather than rounded.
     */
    public static Money ofMajor(BigDecimal major, String currency) {
        Objects.requireNonNull(major, "major");
        int decimals = decimals(currency);
        try {
            return new Money(major.movePointRight(decimals).longValueExact(), currency);
        } catch (ArithmeticException e) {
            throw new IllegalArgumentException(
                    major + " has more than " + decimals + " decimals for " + currency.toUpperCase(), e);
        }
    }

    /** The number of minor-unit decimals for currency, in Stripe's convention. */
    public static int decimals(String currency) {
        String code = currency.toUpperCase();
        if (ZERO_DECIMAL.contains(code)) {
            return 0;
        }
        if (THREE_DECIMAL.contains(code)) {
            return 3;
        }
        return 2;
    }

    public static boolean isZeroDecimal(String currency) {
        return decimals(currency) == 0;
    }

    public BigDecimal toMajor() {
        return BigDecimal.valueOf(amountMinor).movePointLeft(decimals(currency))
                .setScale(decimals(currency), RoundingMode.UNNECESSARY);
    }

    public Money plus(Money other) {
        requireSameCurrency(other);
        return new Money(Math.addExact(amountMinor, other.amountMinor), currency);
    }

    public Money minus(Money other) {
        requireSameCurrency(other);
        return new Money(Math.subtractExact(amountMinor, other.amountMinor), currency);
    }

    public Money times(long quantity) {
        return new Money(Math.multiplyExact(amountMinor, quantity), currency);
    }

    public boolean isPositive() {
        return amountMinor > 0;
    }

    public boolean isNegative() {
        return amountMinor < 0;
    }

    public boolean isZero() {
        return amountMinor == 0;
    }

    @Override
    public int compareTo(Money other) {
        requireSameCurrency(other);
        return Long.compare(amountMinor, other.amountMinor);
    }

    private void requireSameCurrency(Money other) {
        if (!currency.equals(other.currency)) {
            throw new IllegalArgumentException("currency mismatch: " + currency + " vs " + other.currency);
        }
    }

    @Override
    public String toString() {
        return toMajor().toPlainString() + " " + currency;
    }
}
