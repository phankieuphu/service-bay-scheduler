package com.servicebay.billing.domain.payment;

public enum PaymentType {
    /** Paid at booking, before the bill is priced. */
    DEPOSIT,
    /** Pays (part of) the balance once the bill is priced. */
    FINAL
}
