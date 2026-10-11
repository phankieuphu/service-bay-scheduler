package com.servicebay.billing.domain.bill;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.servicebay.billing.domain.money.Money;
import java.time.Instant;
import java.util.List;
import org.junit.jupiter.api.Test;

class BillTest {

    @Test
    void openBillIsUnpricedAndZero() {
        Bill bill = Bill.open(1, 2, 3, "usd");
        assertThat(bill.status()).isEqualTo(BillStatus.OPEN);
        assertThat(bill.currency()).isEqualTo("USD");
        assertThat(bill.isPriced()).isFalse();
        assertThat(bill.total()).isEqualTo(Money.zero("USD"));
    }

    @Test
    void pricingSplitsLinesIntoTheThreeFees() {
        Instant at = Instant.parse("2026-10-01T10:00:00Z");
        Bill priced = Bill.open(1, 2, 3, "USD").price(List.of(
                new BillLine(null, LineKind.SERVICE, 11L, "Brake service", 1, Money.of(15_000, "USD")),
                new BillLine(null, LineKind.SERVICE, 12L, "Inspection", 1, Money.of(5_000, "USD")),
                new BillLine(null, LineKind.MATERIAL, 21L, "Brake pad", 4, Money.of(2_500, "USD")),
                new BillLine(null, LineKind.WARRANTY_FEE, null, "Warranty fee", 1, Money.of(1_000, "USD"))), at);

        assertThat(priced.serviceFee()).isEqualTo(Money.of(20_000, "USD"));
        assertThat(priced.materialCost()).isEqualTo(Money.of(10_000, "USD"));
        assertThat(priced.warrantyFee()).isEqualTo(Money.of(1_000, "USD"));
        assertThat(priced.total()).isEqualTo(Money.of(31_000, "USD"));
        assertThat(priced.status()).isEqualTo(BillStatus.PENDING);
        assertThat(priced.finalizedAt()).isEqualTo(at);
        assertThat(priced.lines()).hasSize(4);
    }

    @Test
    void onlyAnOpenBillCanBePriced() {
        Bill priced = Bill.open(1, 2, 3, "USD").price(List.of(), Instant.EPOCH);
        assertThatThrownBy(() -> priced.price(List.of(), Instant.EPOCH)).isInstanceOf(IllegalStateException.class);
    }

    @Test
    void voidIsFinal() {
        Bill voided = Bill.open(1, 2, 3, "USD").withStatus(BillStatus.VOID);
        assertThatThrownBy(() -> voided.withStatus(BillStatus.PENDING)).isInstanceOf(IllegalStateException.class);
    }

    @Test
    void feesMustMatchTheBillCurrency() {
        Bill open = Bill.open(1, 2, 3, "USD");
        assertThatThrownBy(() -> open.price(List.of(
                new BillLine(null, LineKind.SERVICE, 1L, "x", 1, Money.of(1, "VND"))), Instant.EPOCH))
                .isInstanceOf(IllegalArgumentException.class);
    }

    @Test
    void lineQuantityMustBePositive() {
        assertThatThrownBy(() -> new BillLine(null, LineKind.SERVICE, 1L, "x", 0, Money.of(1, "USD")))
                .isInstanceOf(IllegalArgumentException.class);
    }
}
