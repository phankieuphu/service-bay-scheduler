package com.billing.billing_service.entity;

import java.math.BigDecimal;
import java.time.OffsetDateTime;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.PrePersist;
import jakarta.persistence.PreUpdate;
import jakarta.persistence.Table;

// Maps the `bill` table from postgres/init/06-billing-service.sql.
// total_amount is a generated column in Postgres, so it is not mapped;
// getTotalAmount() computes the same sum in Java.
@Entity
@Table(name = "bill")
public class BillingEntity {

  @Id
  @GeneratedValue(strategy = GenerationType.IDENTITY)
  private Long id;

  @Column(name = "appointment_id", nullable = false, unique = true, updatable = false)
  private Long appointmentId;

  @Column(name = "customer_id", nullable = false, updatable = false)
  private Long customerId;

  @Column(name = "vehicle_id", nullable = false, updatable = false)
  private Long vehicleId;

  @Column(name = "warranty_fee", nullable = false, precision = 12, scale = 2)
  private BigDecimal warrantyFee = BigDecimal.ZERO;

  @Column(name = "service_fee", nullable = false, precision = 12, scale = 2)
  private BigDecimal serviceFee = BigDecimal.ZERO;

  @Column(name = "material_cost", nullable = false, precision = 12, scale = 2)
  private BigDecimal materialCost = BigDecimal.ZERO;

  @Enumerated(EnumType.STRING)
  @Column(nullable = false, length = 20)
  private BillStatus status = BillStatus.PENDING;

  @Column(name = "created_at", nullable = false, updatable = false)
  private OffsetDateTime createdAt;

  @Column(name = "updated_at", nullable = false)
  private OffsetDateTime updatedAt;

  protected BillingEntity() {
  }

  public BillingEntity(Long appointmentId, Long customerId, Long vehicleId,
      BigDecimal warrantyFee, BigDecimal serviceFee, BigDecimal materialCost) {
    this.appointmentId = appointmentId;
    this.customerId = customerId;
    this.vehicleId = vehicleId;
    this.warrantyFee = warrantyFee;
    this.serviceFee = serviceFee;
    this.materialCost = materialCost;
  }

  @PrePersist
  void onCreate() {
    OffsetDateTime now = OffsetDateTime.now();
    this.createdAt = now;
    this.updatedAt = now;
  }

  @PreUpdate
  void onUpdate() {
    this.updatedAt = OffsetDateTime.now();
  }

  public BigDecimal getTotalAmount() {
    return warrantyFee.add(serviceFee).add(materialCost);
  }

  public Long getId() {
    return id;
  }

  public Long getAppointmentId() {
    return appointmentId;
  }

  public Long getCustomerId() {
    return customerId;
  }

  public Long getVehicleId() {
    return vehicleId;
  }

  public BigDecimal getWarrantyFee() {
    return warrantyFee;
  }

  public BigDecimal getServiceFee() {
    return serviceFee;
  }

  public BigDecimal getMaterialCost() {
    return materialCost;
  }

  public BillStatus getStatus() {
    return status;
  }

  public void setStatus(BillStatus status) {
    this.status = status;
  }

  public OffsetDateTime getCreatedAt() {
    return createdAt;
  }

  public OffsetDateTime getUpdatedAt() {
    return updatedAt;
  }
}
