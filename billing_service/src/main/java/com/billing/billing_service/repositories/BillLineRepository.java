package com.billing.billing_service.repositories;

import java.util.List;

import org.springframework.data.jpa.repository.JpaRepository;

import com.billing.billing_service.entity.BillLineEntity;

public interface BillLineRepository extends JpaRepository<BillLineEntity, Long> {

  List<BillLineEntity> findByBillIdOrderByIdAsc(Long billId);
}
