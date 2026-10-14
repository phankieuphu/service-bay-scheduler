package com.payment.payment_service.gateway;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.math.BigDecimal;

import org.junit.jupiter.api.Test;

class MoneyTest {

  @Test
  void twoDecimalCurrencyIsSentInCents() {
    assertEquals(12345L, Money.toMinorUnits(new BigDecimal("123.45"), "USD"));
    assertEquals(500L, Money.toMinorUnits(new BigDecimal("5"), "eur"));
  }

  @Test
  void zeroDecimalCurrencyIsSentAsIs() {
    assertEquals(150000L, Money.toMinorUnits(new BigDecimal("150000.00"), "VND"));
  }

  @Test
  void rejectsAmountsThatDontFitTheCurrency() {
    assertThrows(IllegalArgumentException.class, () -> Money.toMinorUnits(new BigDecimal("1.005"), "usd"));
    assertThrows(IllegalArgumentException.class, () -> Money.toMinorUnits(new BigDecimal("10.50"), "vnd"));
  }

  @Test
  void rejectsBadCurrencyCodes() {
    assertThrows(IllegalArgumentException.class, () -> Money.normalizeCurrency("dollars"));
    assertThrows(IllegalArgumentException.class, () -> Money.normalizeCurrency(null));
  }
}
