package com.servicebay.billing.domain.deposit;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.servicebay.billing.domain.money.Money;
import java.math.BigDecimal;
import org.junit.jupiter.api.Test;

class DepositPolicyTest {

    private final Money estimate = Money.of(10_550, "USD"); // 105.50

    @Test
    void percentRoundsUpToAWholeMinorUnit() {
        assertThat(DepositPolicy.parse("percent", BigDecimal.valueOf(20)).amountFor(estimate)).isEqualTo(Money.of(2_110, "USD"));
        assertThat(DepositPolicy.parse("PERCENT", new BigDecimal("33.3")).amountFor(Money.of(100, "USD")))
                .isEqualTo(Money.of(34, "USD"));
        assertThat(DepositPolicy.parse("PERCENT", BigDecimal.valueOf(10)).amountFor(Money.of(150_005, "VND")))
                .isEqualTo(Money.of(15_001, "VND"));
    }

    @Test
    void fixedIsCappedAtTheEstimate() {
        assertThat(DepositPolicy.parse("FIXED", new BigDecimal("20.00")).amountFor(estimate)).isEqualTo(Money.of(2_000, "USD"));
        assertThat(DepositPolicy.parse("FIXED", new BigDecimal("500")).amountFor(estimate)).isEqualTo(estimate);
    }

    @Test
    void fullIsTheEstimateAndAFreeBookingNeedsNone() {
        assertThat(DepositPolicy.parse("FULL", null).amountFor(estimate)).isEqualTo(estimate);
        assertThat(DepositPolicy.parse("PERCENT", BigDecimal.TEN).amountFor(Money.zero("USD")).isZero()).isTrue();
    }

    @Test
    void rejectsNonsense() {
        assertThatThrownBy(() -> DepositPolicy.parse("PERCENT", BigDecimal.valueOf(150))).isInstanceOf(IllegalArgumentException.class);
        assertThatThrownBy(() -> DepositPolicy.parse("FIXED", BigDecimal.ZERO)).isInstanceOf(IllegalArgumentException.class);
        assertThatThrownBy(() -> DepositPolicy.parse("SOMETIMES", BigDecimal.ONE)).isInstanceOf(IllegalArgumentException.class);
    }
}
