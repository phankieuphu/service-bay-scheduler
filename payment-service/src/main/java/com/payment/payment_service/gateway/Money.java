package com.payment.payment_service.gateway;

import java.math.BigDecimal;
import java.util.Locale;
import java.util.Set;

// Converts between the numeric(12,2) amounts the databases store and the integer
// minor units Stripe expects. Most currencies use 2 decimals (cents), but Stripe
// treats some as zero-decimal (VND, JPY, KRW, ...): 50000 VND is sent as 50000,
// not 5000000.
public final class Money {

  // https://docs.stripe.com/currencies#zero-decimal
  private static final Set<String> ZERO_DECIMAL = Set.of(
      "bif", "clp", "djf", "gnf", "jpy", "kmf", "krw", "mga",
      "pyg", "rwf", "ugx", "vnd", "vuv", "xaf", "xof", "xpf");

  private Money() {
  }

  public static String normalizeCurrency(String currency) {
    if (currency == null || !currency.matches("[A-Za-z]{3}")) {
      throw new IllegalArgumentException("currency must be a 3-letter ISO code");
    }
    return currency.toLowerCase(Locale.ROOT);
  }

  public static int fractionDigits(String currency) {
    return ZERO_DECIMAL.contains(normalizeCurrency(currency)) ? 0 : 2;
  }

  // Rejects amounts that don't fit the currency exactly (e.g. 10.50 VND) rather
  // than silently rounding what the customer is charged.
  public static long toMinorUnits(BigDecimal amount, String currency) {
    try {
      return amount.movePointRight(fractionDigits(currency)).longValueExact();
    } catch (ArithmeticException e) {
      throw new IllegalArgumentException(
          "amount " + amount.toPlainString() + " is not representable in " + normalizeCurrency(currency));
    }
  }
}
