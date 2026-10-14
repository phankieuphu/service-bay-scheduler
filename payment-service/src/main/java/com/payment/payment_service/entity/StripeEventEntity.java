package com.payment.payment_service.entity;

import java.time.OffsetDateTime;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Id;
import jakarta.persistence.Table;

// Maps `stripe_event`, the webhook dedupe log. Rows are only inserted through
// StripeEventRepository.markProcessed.
@Entity
@Table(name = "stripe_event")
public class StripeEventEntity {

  @Id
  private String id;

  @Column(nullable = false)
  private String type;

  @Column(name = "received_at", nullable = false)
  private OffsetDateTime receivedAt;

  protected StripeEventEntity() {
  }

  public String getId() {
    return id;
  }

  public String getType() {
    return type;
  }

  public OffsetDateTime getReceivedAt() {
    return receivedAt;
  }
}
