package com.billing.billing_service.repositories;

import java.math.BigDecimal;
import java.util.List;
import java.util.Optional;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import com.billing.billing_service.entity.PaymentEntity;
import com.billing.billing_service.entity.PaymentStatus;

@Repository
public interface PaymentRepository extends JpaRepository<PaymentEntity, Long> {

  List<PaymentEntity> findByBillIdOrderByCreatedAtAsc(Long billId);

  @Query("select coalesce(sum(p.amount), 0) from PaymentEntity p "
      + "where p.billId = :billId and p.status = com.billing.billing_service.entity.PaymentStatus.SUCCEEDED")
  BigDecimal sumSucceededAmount(@Param("billId") Long billId);

  // Card checkouts still waiting on payment-service. They count against the
  // outstanding balance until they succeed or fail.
  @Query("select coalesce(sum(p.amount), 0) from PaymentEntity p "
      + "where p.billId = :billId and p.status = com.billing.billing_service.entity.PaymentStatus.PENDING")
  BigDecimal sumPendingAmount(@Param("billId") Long billId);

  boolean existsByBillIdAndStatus(Long billId, PaymentStatus status);

  // Only the bill id, so the caller can lock the bill before loading the payment itself.
  @Query("select p.billId from PaymentEntity p where p.id = :id")
  Optional<Long> findBillIdById(@Param("id") Long id);
}
