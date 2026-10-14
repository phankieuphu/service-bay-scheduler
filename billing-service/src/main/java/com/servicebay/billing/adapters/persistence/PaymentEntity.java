package com.servicebay.billing.adapters.persistence;

import com.servicebay.billing.domain.payment.PaymentStatus;
import com.servicebay.billing.domain.payment.PaymentType;
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

@Entity
@Table(name = "payment")
class PaymentEntity {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    Long id;

    @Column(name = "bill_id", nullable = false, updatable = false)
    long billId;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 20, updatable = false)
    PaymentType type;

    @Column(name = "amount_minor", nullable = false, updatable = false)
    long amountMinor;

    @JdbcTypeCode(SqlTypes.CHAR)
    @Column(nullable = false, length = 3, updatable = false)
    String currency;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 20)
    PaymentStatus status;

    @Column(name = "stripe_payment_intent_id")
    String stripePaymentIntentId;

    @Column(name = "idempotency_key", nullable = false, updatable = false)
    String idempotencyKey;

    @Column(name = "failure_code", length = 100)
    String failureCode;

    @Column(name = "failure_message", length = 500)
    String failureMessage;

    @Column(name = "card_brand", length = 30)
    String cardBrand;

    @Column(name = "card_last4", length = 4)
    String cardLast4;

    @Column(name = "succeeded_at")
    Instant succeededAt;

    @Version
    long version;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    Instant createdAt;

    @UpdateTimestamp
    @Column(name = "updated_at", nullable = false)
    Instant updatedAt;
}
