package com.servicebay.billing.adapters.persistence;

import jakarta.persistence.LockModeType;
import java.util.List;
import java.util.Optional;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Lock;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

/** Spring Data interfaces. Only the adapters in this package use them. */
final class SpringDataRepositories {

    private SpringDataRepositories() {
    }

    interface BillJpa extends JpaRepository<BillEntity, Long> {

        @Lock(LockModeType.PESSIMISTIC_WRITE)
        @Query("select b from BillEntity b where b.id = :id")
        Optional<BillEntity> findByIdForUpdate(@Param("id") long id);

        @Query("select b from BillEntity b where b.appointmentId = :appointmentId "
                + "and b.status <> com.servicebay.billing.domain.bill.BillStatus.VOID")
        Optional<BillEntity> findLiveByAppointment(@Param("appointmentId") long appointmentId);

        List<BillEntity> findByCustomerIdOrderByCreatedAtDesc(long customerId);
    }

    interface BillLineJpa extends JpaRepository<BillLineEntity, Long> {

        List<BillLineEntity> findByBillIdOrderById(long billId);
    }

    interface PaymentJpa extends JpaRepository<PaymentEntity, Long> {

        Optional<PaymentEntity> findByIdempotencyKey(String idempotencyKey);

        Optional<PaymentEntity> findByStripePaymentIntentId(String stripePaymentIntentId);

        List<PaymentEntity> findByBillIdOrderById(long billId);
    }

    interface RefundJpa extends JpaRepository<RefundEntity, Long> {

        Optional<RefundEntity> findByStripeRefundId(String stripeRefundId);

        List<RefundEntity> findByPaymentIdOrderById(long paymentId);

        @Query("select r from RefundEntity r where r.paymentId in "
                + "(select p.id from PaymentEntity p where p.billId = :billId) order by r.id")
        List<RefundEntity> findByBillId(@Param("billId") long billId);
    }

    interface OutboxJpa extends JpaRepository<OutboxMessageEntity, Long> {
    }
}
