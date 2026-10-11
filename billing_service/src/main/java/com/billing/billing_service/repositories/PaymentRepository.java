package com.billing.billing_service.repositories;

import java.math.BigDecimal;
import java.util.List;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import com.billing.billing_service.entity.PaymentEntity;

@Repository
public interface PaymentRepository extends JpaRepository<PaymentEntity, Long> {

  List<PaymentEntity> findByBillIdOrderByCreatedAtAsc(Long billId);

  @Query("select coalesce(sum(p.amount), 0) from PaymentEntity p "
      + "where p.billId = :billId and p.status = com.billing.billing_service.entity.PaymentStatus.SUCCEEDED")
  BigDecimal sumSucceededAmount(@Param("billId") Long billId);
}
