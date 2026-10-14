package com.payment.payment_service.repositories;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.transaction.annotation.Transactional;

import com.payment.payment_service.entity.StripeEventEntity;

public interface StripeEventRepository extends JpaRepository<StripeEventEntity, String> {

  // Returns 1 the first time an event id is seen and 0 afterwards. Must run in
  // the same transaction as the state change, so a rollback un-marks the event
  // and Stripe's retry gets processed.
  @Transactional
  @Modifying
  @Query(value = "insert into stripe_event (id, type) values (:id, :type) on conflict (id) do nothing",
      nativeQuery = true)
  int markProcessed(@Param("id") String id, @Param("type") String type);
}
