package com.servicebay.billing.domain.bill;

import com.servicebay.billing.domain.money.Money;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;

/**
 * The bill formula (business-requirement §6):
 * <pre>
 * total = warranty fee (only if the warranty had expired, or there is none, on the service date)
 *       + service fees + material costs
 * </pre>
 * The warranty decision is made by the caller from vehicle-service's answer;
 * this only turns it into lines.
 */
public final class Pricing {

    private Pricing() {
    }

    /**
     * @param warrantyEndDate the vehicle's warranty end date, null if there is none on record
     * @param inWarranty vehicle-service's verdict for the service date (the last day is covered)
     * @param warrantyFee what to charge when not in warranty, in the bill's currency
     */
    public static List<BillLine> lines(List<BillLine> services, List<BillLine> materials,
            boolean inWarranty, LocalDate warrantyEndDate, Money warrantyFee) {
        List<BillLine> lines = new ArrayList<>();
        if (!inWarranty && warrantyFee.isPositive()) {
            String why = warrantyEndDate == null ? "no warranty on record" : "warranty ended " + warrantyEndDate;
            lines.add(new BillLine(null, LineKind.WARRANTY_FEE, null, "Warranty/expiration fee (" + why + ")", 1, warrantyFee));
        }
        lines.addAll(services);
        lines.addAll(materials);
        return lines;
    }
}
