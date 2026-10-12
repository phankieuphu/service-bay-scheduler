package com.servicebay.billing.adapters.persistence;

import com.servicebay.billing.domain.bill.BillStatus;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import jakarta.persistence.Version;
import java.time.Instant;
import org.hibernate.annotations.CreationTimestamp;
import org.hibernate.annotations.JdbcTypeCode;
import org.hibernate.annotations.UpdateTimestamp;
import org.hibernate.type.SqlTypes;

// total_minor is a generated column; the domain computes the same sum, so
// it isn't mapped.
@Entity
@Table(name = "bill")
class BillEntity {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    Long id;

    @Column(name = "appointment_id", nullable = false, updatable = false)
    long appointmentId;

    @Column(name = "customer_id", nullable = false, updatable = false)
    long customerId;

    @Column(name = "vehicle_id", nullable = false, updatable = false)
    long vehicleId;

    @JdbcTypeCode(SqlTypes.CHAR)
    @Column(nullable = false, length = 3, updatable = false)
    String currency;

    @Column(name = "warranty_fee_minor", nullable = false)
    long warrantyFeeMinor;

    @Column(name = "service_fee_minor", nullable = false)
    long serviceFeeMinor;

    @Column(name = "material_cost_minor", nullable = false)
    long materialCostMinor;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 20)
    BillStatus status;

    @Column(name = "finalized_at")
    Instant finalizedAt;

    @Version
    long version;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    Instant createdAt;

    @UpdateTimestamp
    @Column(name = "updated_at", nullable = false)
    Instant updatedAt;
}
