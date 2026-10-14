package com.payment.payment_service.repositories;

import java.util.List;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.transaction.annotation.Transactional;

import com.payment.payment_service.entity.OutboxMessageEntity;

public interface OutboxRepository extends JpaRepository<OutboxMessageEntity, Long> {

  // Oldest first, so per-key ordering survives the relay.
  @Query(value = "select * from outbox_message where published_at is null order by id limit :limit",
      nativeQuery = true)
  List<OutboxMessageEntity> fetchUnpublished(@Param("limit") int limit);

  @Transactional
  @Modifying
  @Query(value = "update outbox_message set published_at = now() where id in (:ids)", nativeQuery = true)
  int markPublished(@Param("ids") List<Long> ids);
}
