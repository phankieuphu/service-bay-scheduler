package com.payment.payment_service.entity;

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

// Maps the `payment` table from postgres/init/11-payment-service.sql:
// one row per billing-service card payment, linked to one Stripe PaymentIntent.
@Entity
@Table(name = "payment")
public class PaymentEntity {

  @Id
  @GeneratedValue(strategy = GenerationType.IDENTITY)
  private Long id;

  @Column(name = "bill_id", nullable = false, updatable = false)
  private Long billId;

  @Column(name = "billing_payment_id", nullable = false, unique = true, updatable = false)
  private Long billingPaymentId;

  @Column(name = "customer_id", nullable = false, updatable = false)
  private Long customerId;

  @Column(nullable = false, precision = 12, scale = 2, updatable = false)
  private BigDecimal amount;

  @Column(nullable = false, length = 3, updatable = false)
  private String currency;

  @Column(name = "stripe_payment_intent_id", unique = true)
  private String stripePaymentIntentId;

  @Enumerated(EnumType.STRING)
  @Column(nullable = false, length = 20)
  private PaymentStatus status = PaymentStatus.PENDING;

  @Column(name = "failure_reason", length = 500)
  private String failureReason;

  @Column(name = "created_at", nullable = false, updatable = false)
  private OffsetDateTime createdAt;

  @Column(name = "updated_at", nullable = false)
  private OffsetDateTime updatedAt;

  protected PaymentEntity() {
  }

  public PaymentEntity(Long billId, Long billingPaymentId, Long customerId, BigDecimal amount, String currency) {
    this.billId = billId;
    this.billingPaymentId = billingPaymentId;
    this.customerId = customerId;
    this.amount = amount;
    this.currency = currency;
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

  public Long getId() {
    return id;
  }

  public Long getBillId() {
    return billId;
  }

  public Long getBillingPaymentId() {
    return billingPaymentId;
  }

  public Long getCustomerId() {
    return customerId;
  }

  public BigDecimal getAmount() {
    return amount;
  }

  public String getCurrency() {
    return currency;
  }

  public String getStripePaymentIntentId() {
    return stripePaymentIntentId;
  }

  public void setStripePaymentIntentId(String stripePaymentIntentId) {
    this.stripePaymentIntentId = stripePaymentIntentId;
  }

  public PaymentStatus getStatus() {
    return status;
  }

  public void setStatus(PaymentStatus status) {
    this.status = status;
  }

  public String getFailureReason() {
    return failureReason;
  }

  public void setFailureReason(String failureReason) {
    this.failureReason = failureReason == null || failureReason.length() <= 500
        ? failureReason
        : failureReason.substring(0, 500);
  }

  public OffsetDateTime getCreatedAt() {
    return createdAt;
  }

  public OffsetDateTime getUpdatedAt() {
    return updatedAt;
  }
}
