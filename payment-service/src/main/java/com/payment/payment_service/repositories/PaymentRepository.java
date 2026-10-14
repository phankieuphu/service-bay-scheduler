package com.payment.payment_service.repositories;

import java.util.Optional;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Lock;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

import com.payment.payment_service.entity.PaymentEntity;

import jakarta.persistence.LockModeType;

public interface PaymentRepository extends JpaRepository<PaymentEntity, Long> {

  Optional<PaymentEntity> findByBillingPaymentId(Long billingPaymentId);

  // SELECT ... FOR UPDATE: a webhook and a cancel request for the same payment
  // serialize here, so a terminal status is never overwritten.
  @Lock(LockModeType.PESSIMISTIC_WRITE)
  @Query("select p from PaymentEntity p where p.id = :id")
  Optional<PaymentEntity> findByIdForUpdate(@Param("id") Long id);

  @Lock(LockModeType.PESSIMISTIC_WRITE)
  @Query("select p from PaymentEntity p where p.stripePaymentIntentId = :intentId")
  Optional<PaymentEntity> findByStripePaymentIntentIdForUpdate(@Param("intentId") String intentId);
}
