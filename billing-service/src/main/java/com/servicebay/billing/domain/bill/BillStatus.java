package com.servicebay.billing.domain.bill;

/** A bill's lifecycle. Only OPEN bills can be priced; VOID is final. */
public enum BillStatus {
    /** Created by a deposit; not priced yet. */
    OPEN,
    /** Priced; nothing paid toward it yet. */
    PENDING,
    /** Priced; part of the total is paid (a deposit counts). */
    PARTIALLY_PAID,
    PAID,
    VOID
}
