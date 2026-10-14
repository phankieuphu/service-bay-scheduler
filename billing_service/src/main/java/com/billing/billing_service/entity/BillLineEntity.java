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
import jakarta.persistence.Table;

// Maps `bill_line`: one itemized line of a bill created by checkout.
@Entity
@Table(name = "bill_line")
public class BillLineEntity {

  @Id
  @GeneratedValue(strategy = GenerationType.IDENTITY)
  private Long id;

  @Column(name = "bill_id", nullable = false, updatable = false)
  private Long billId;

  @Enumerated(EnumType.STRING)
  @Column(nullable = false, length = 20, updatable = false)
  private BillLineKind kind;

  @Column(nullable = false, updatable = false)
  private String description;

  @Column(nullable = false, updatable = false)
  private int quantity;

  @Column(name = "unit_price", nullable = false, precision = 12, scale = 2, updatable = false)
  private BigDecimal unitPrice;

  @Column(nullable = false, precision = 12, scale = 2, updatable = false)
  private BigDecimal amount;

  @Column(name = "created_at", nullable = false, updatable = false)
  private OffsetDateTime createdAt;

  protected BillLineEntity() {
  }

  public BillLineEntity(Long billId, BillLineKind kind, String description, int quantity, BigDecimal unitPrice) {
    this.billId = billId;
    this.kind = kind;
    this.description = description;
    this.quantity = quantity;
    this.unitPrice = unitPrice;
    this.amount = unitPrice.multiply(BigDecimal.valueOf(quantity));
  }

  @PrePersist
  void onCreate() {
    this.createdAt = OffsetDateTime.now();
  }

  public Long getId() {
    return id;
  }

  public Long getBillId() {
    return billId;
  }

  public BillLineKind getKind() {
    return kind;
  }

  public String getDescription() {
    return description;
  }

  public int getQuantity() {
    return quantity;
  }

  public BigDecimal getUnitPrice() {
    return unitPrice;
  }

  public BigDecimal getAmount() {
    return amount;
  }
}
