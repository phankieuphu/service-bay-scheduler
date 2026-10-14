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

  @Enumerated(EnumType.STRING)
  @Column(nullable = false, length = 30)
  private PaymentMethod method;

  @Column(nullable = false, precision = 12, scale = 2)
  private BigDecimal amount;

  @Enumerated(EnumType.STRING)
  @Column(nullable = false, length = 20)
  private PaymentStatus status = PaymentStatus.PENDING;

  // payment-service's payment id; set for CARD payments once the checkout is created.
  @Column(name = "external_payment_id", unique = true)
  private Long externalPaymentId;

  @Column(name = "failure_reason", length = 500)
  private String failureReason;

  @Column(name = "paid_at")
  private OffsetDateTime paidAt;

  @Column(name = "created_at", nullable = false, updatable = false)
  private OffsetDateTime createdAt;

  protected PaymentEntity() {
  }

  public PaymentEntity(Long billId, PaymentType type, PaymentMethod method, BigDecimal amount) {
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
    this.failureReason = null;
    this.paidAt = OffsetDateTime.now();
  }

  public void markFailed(String reason) {
    this.status = PaymentStatus.FAILED;
    this.failureReason = reason == null || reason.length() <= 500 ? reason : reason.substring(0, 500);
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

  public PaymentMethod getMethod() {
    return method;
  }

  public Long getExternalPaymentId() {
    return externalPaymentId;
  }

  public void setExternalPaymentId(Long externalPaymentId) {
    this.externalPaymentId = externalPaymentId;
  }

  public String getFailureReason() {
    return failureReason;
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
