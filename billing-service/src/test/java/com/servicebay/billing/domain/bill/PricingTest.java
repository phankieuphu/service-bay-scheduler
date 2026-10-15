package com.servicebay.billing.domain.bill;

import static org.assertj.core.api.Assertions.assertThat;

import com.servicebay.billing.domain.money.Money;
import java.time.LocalDate;
import java.util.List;
import org.junit.jupiter.api.Test;

class PricingTest {

    private final List<BillLine> services = List.of(new BillLine(null, LineKind.SERVICE, 1L, "Oil change", 1, Money.of(4_500, "USD")));
    private final List<BillLine> materials = List.of(new BillLine(null, LineKind.MATERIAL, 9L, "Oil filter", 2, Money.of(1_250, "USD")));
    private final Money fee = Money.of(5_000, "USD");

    @Test
    void inWarrantyNoFee() {
        List<BillLine> lines = Pricing.lines(services, materials, true, LocalDate.of(2029, 1, 12), fee);
        assertThat(lines).extracting(BillLine::kind).containsExactly(LineKind.SERVICE, LineKind.MATERIAL);
    }

    @Test
    void expiredWarrantySaysWhenItEnded() {
        List<BillLine> lines = Pricing.lines(services, materials, false, LocalDate.of(2026, 9, 27), fee);
        assertThat(lines.get(0).kind()).isEqualTo(LineKind.WARRANTY_FEE);
        assertThat(lines.get(0).name()).isEqualTo("Warranty/expiration fee (warranty ended 2026-09-27)");
        assertThat(lines.get(0).total()).isEqualTo(fee);
    }

    @Test
    void noWarrantyOnRecord() {
        assertThat(Pricing.lines(services, List.of(), false, null, fee).get(0).name()).contains("no warranty on record");
    }

    @Test
    void aZeroFeeAddsNoLine() {
        assertThat(Pricing.lines(services, List.of(), false, null, Money.zero("USD"))).hasSize(1);
    }

    @Test
    void pricedBillTotalsMatchTheLines() {
        Bill bill = Bill.open(1, 2, 3, "USD").price(Pricing.lines(services, materials, false, null, fee), java.time.Instant.now());
        assertThat(bill.warrantyFee()).isEqualTo(fee);
        assertThat(bill.serviceFee()).isEqualTo(Money.of(4_500, "USD"));
        assertThat(bill.materialCost()).isEqualTo(Money.of(2_500, "USD"));
        assertThat(bill.total()).isEqualTo(Money.of(12_000, "USD"));
    }
}
