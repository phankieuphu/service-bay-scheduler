package com.servicebay.billing.adapters.persistence;

import com.servicebay.billing.domain.bill.LineKind;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import java.time.Instant;
import org.hibernate.annotations.CreationTimestamp;

// Lines are written once, when the bill is priced, and never changed.
@Entity
@Table(name = "bill_line")
class BillLineEntity {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    Long id;

    @Column(name = "bill_id", nullable = false, updatable = false)
    long billId;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 20, updatable = false)
    LineKind kind;

    @Column(name = "source_id", updatable = false)
    Long sourceId;

    @Column(nullable = false, updatable = false)
    String name;

    @Column(nullable = false, updatable = false)
    int quantity;

    @Column(name = "unit_price_minor", nullable = false, updatable = false)
    long unitPriceMinor;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    Instant createdAt;
}
