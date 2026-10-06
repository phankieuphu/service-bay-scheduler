package com.billing.billing_service.repositories;

import java.util.List;
import java.util.Optional;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Lock;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import com.billing.billing_service.entity.BillingEntity;

import jakarta.persistence.LockModeType;

public interface BillingRepository extends JpaRepository<BillingEntity, Long> {

  List<BillingEntity> findByCustomerIdOrderByCreatedAtDesc(Long customerId);

  boolean existsByAppointmentId(Long appointmentId);

  // SELECT ... FOR UPDATE: serializes concurrent payments against the same bill
  // so the paid-sum check and the status update can't interleave.
  @Lock(LockModeType.PESSIMISTIC_WRITE)
  @Query("select b from BillingEntity b where b.id = :id")
  Optional<BillingEntity> findByIdForUpdate(@Param("id") Long id);
}
