package com.servicebay.billing.domain.bill;

import com.servicebay.billing.domain.money.Money;
import java.util.Objects;

/**
 * One charged item, copied from its source (catalog service, consumed
 * material, warranty fee) when the bill is priced.
 *
 * @param sourceId the service_id or material_id in its owning service; null for a warranty fee
 */
public record BillLine(Long id, LineKind kind, Long sourceId, String name, int quantity, Money unitPrice) {

    public BillLine {
        Objects.requireNonNull(kind, "kind");
        Objects.requireNonNull(name, "name");
        Objects.requireNonNull(unitPrice, "unitPrice");
        if (quantity <= 0) {
            throw new IllegalArgumentException("quantity must be positive");
        }
        if (unitPrice.isNegative()) {
            throw new IllegalArgumentException("unit price must not be negative");
        }
    }

    public Money total() {
        return unitPrice.times(quantity);
    }
}
