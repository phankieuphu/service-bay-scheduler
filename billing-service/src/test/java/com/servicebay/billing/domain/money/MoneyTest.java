package com.servicebay.billing.domain.money;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.math.BigDecimal;
import org.junit.jupiter.api.Test;

class MoneyTest {

    @Test
    void twoDecimalCurrencyConvertsBothWays() {
        Money m = Money.ofMajor(new BigDecimal("12.34"), "usd");
        assertThat(m.amountMinor()).isEqualTo(1234);
        assertThat(m.currency()).isEqualTo("USD");
        assertThat(m.toMajor()).isEqualByComparingTo("12.34");
        assertThat(m).hasToString("12.34 USD");
    }

    @Test
    void zeroDecimalCurrencyHasNoMinorUnits() {
        Money vnd = Money.ofMajor(new BigDecimal("150000"), "VND");
        assertThat(vnd.amountMinor()).isEqualTo(150000);
        assertThat(Money.isZeroDecimal("VND")).isTrue();
        assertThat(Money.isZeroDecimal("JPY")).isTrue();
        assertThat(vnd).hasToString("150000 VND");
    }

    @Test
    void threeDecimalCurrency() {
        assertThat(Money.ofMajor(new BigDecimal("1.234"), "KWD").amountMinor()).isEqualTo(1234);
    }

    @Test
    void rejectsMorePrecisionThanTheCurrencyHas() {
        assertThatThrownBy(() -> Money.ofMajor(new BigDecimal("1.5"), "VND"))
                .isInstanceOf(IllegalArgumentException.class);
        assertThatThrownBy(() -> Money.ofMajor(new BigDecimal("1.234"), "USD"))
                .isInstanceOf(IllegalArgumentException.class);
    }

    @Test
    void arithmeticRequiresTheSameCurrency() {
        Money a = Money.of(1000, "USD");
        assertThat(a.plus(Money.of(250, "USD"))).isEqualTo(Money.of(1250, "USD"));
        assertThat(a.minus(Money.of(1250, "USD")).isNegative()).isTrue();
        assertThat(a.times(3)).isEqualTo(Money.of(3000, "USD"));
        assertThatThrownBy(() -> a.plus(Money.of(1, "VND"))).isInstanceOf(IllegalArgumentException.class);
    }

    @Test
    void overflowThrowsInsteadOfWrapping() {
        assertThatThrownBy(() -> Money.of(Long.MAX_VALUE, "USD").plus(Money.of(1, "USD")))
                .isInstanceOf(ArithmeticException.class);
    }

    @Test
    void rejectsUnknownCurrency() {
        assertThatThrownBy(() -> Money.of(1, "XYZ")).isInstanceOf(IllegalArgumentException.class);
    }
}
