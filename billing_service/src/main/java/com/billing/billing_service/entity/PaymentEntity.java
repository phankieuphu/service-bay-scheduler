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

// Maps the `payment` table. A deposit and the final payment are both rows here.
@Entity
@Table(name = "payment")
public class PaymentEntity {

  @Id
  @GeneratedValue(strategy = GenerationType.IDENTITY)
  private Long id;

  @Column(name = "bill_id", nullable = false, updatable = false)
  private Long billId;

  @Enumerated(EnumType.STRING)
  @Column(nullable = false, length = 20)
  private PaymentType type;

  @Column(nullable = false, length = 30)
  private String method;

  @Column(nullable = false, precision = 12, scale = 2)
  private BigDecimal amount;

  @Enumerated(EnumType.STRING)
  @Column(nullable = false, length = 20)
  private PaymentStatus status = PaymentStatus.PENDING;

  @Column(name = "paid_at")
  private OffsetDateTime paidAt;

  @Column(name = "created_at", nullable = false, updatable = false)
  private OffsetDateTime createdAt;

  protected PaymentEntity() {
  }

  public PaymentEntity(Long billId, PaymentType type, String method, BigDecimal amount) {
    this.billId = billId;
    this.type = type;
    this.method = method;
    this.amount = amount;
  }

  @PrePersist
  void onCreate() {
    this.createdAt = OffsetDateTime.now();
  }

  public void markSucceeded() {
    this.status = PaymentStatus.SUCCEEDED;
    this.paidAt = OffsetDateTime.now();
  }

  public Long getId() {
    return id;
  }

  public Long getBillId() {
    return billId;
  }

  public PaymentType getType() {
    return type;
  }

  public String getMethod() {
    return method;
  }

  public BigDecimal getAmount() {
    return amount;
  }

  public PaymentStatus getStatus() {
    return status;
  }

  public OffsetDateTime getPaidAt() {
    return paidAt;
  }

  public OffsetDateTime getCreatedAt() {
    return createdAt;
  }
}
