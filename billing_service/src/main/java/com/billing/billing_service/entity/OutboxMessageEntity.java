package com.billing.billing_service.entity;

import java.time.OffsetDateTime;

import org.hibernate.annotations.JdbcTypeCode;
import org.hibernate.type.SqlTypes;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.PrePersist;
import jakarta.persistence.Table;

// Maps `outbox_message`: written in the same transaction as the state change it
// describes, published to Kafka later by OutboxRelay.
@Entity
@Table(name = "outbox_message")
public class OutboxMessageEntity {

  @Id
  @GeneratedValue(strategy = GenerationType.IDENTITY)
  private Long id;

  @Column(nullable = false, updatable = false)
  private String topic;

  @Column(name = "message_key", nullable = false, updatable = false)
  private String messageKey;

  @JdbcTypeCode(SqlTypes.JSON)
  @Column(nullable = false, updatable = false, columnDefinition = "jsonb")
  private String payload;

  @Column(name = "created_at", nullable = false, updatable = false)
  private OffsetDateTime createdAt;

  @Column(name = "published_at")
  private OffsetDateTime publishedAt;

  protected OutboxMessageEntity() {
  }

  public OutboxMessageEntity(String topic, String messageKey, String payload) {
    this.topic = topic;
    this.messageKey = messageKey;
    this.payload = payload;
  }

  @PrePersist
  void onCreate() {
    this.createdAt = OffsetDateTime.now();
  }

  public Long getId() {
    return id;
  }

  public String getTopic() {
    return topic;
  }

  public String getMessageKey() {
    return messageKey;
  }

  public String getPayload() {
    return payload;
  }

  public OffsetDateTime getPublishedAt() {
    return publishedAt;
  }
}
