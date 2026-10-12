package com.servicebay.billing.adapters.persistence;

import com.servicebay.billing.adapters.persistence.SpringDataRepositories.RefundJpa;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.payment.Refund;
import com.servicebay.billing.domain.ports.ConflictException;
import com.servicebay.billing.domain.ports.RefundRepository;
import java.util.List;
import java.util.Optional;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.orm.ObjectOptimisticLockingFailureException;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Component
class JpaRefundRepository implements RefundRepository {

    private final RefundJpa refunds;

    JpaRefundRepository(RefundJpa refunds) {
        this.refunds = refunds;
    }

    @Override
    @Transactional
    public Refund save(Refund refund) {
        try {
            return toDomain(refunds.saveAndFlush(toEntity(refund)));
        } catch (DataIntegrityViolationException e) {
            throw new ConflictException("refund conflicts with an existing one", e);
        } catch (ObjectOptimisticLockingFailureException e) {
            throw new ConflictException("refund " + refund.id() + " was changed by someone else", e);
        }
    }

    @Override
    @Transactional(readOnly = true)
    public Optional<Refund> findByStripeRefundId(String stripeRefundId) {
        return refunds.findByStripeRefundId(stripeRefundId).map(JpaRefundRepository::toDomain);
    }

    @Override
    @Transactional(readOnly = true)
    public List<Refund> findByPayment(long paymentId) {
        return refunds.findByPaymentIdOrderById(paymentId).stream().map(JpaRefundRepository::toDomain).toList();
    }

    @Override
    @Transactional(readOnly = true)
    public List<Refund> findByBill(long billId) {
        return refunds.findByBillId(billId).stream().map(JpaRefundRepository::toDomain).toList();
    }

    private static Refund toDomain(RefundEntity e) {
        return new Refund(e.id, e.paymentId, Money.of(e.amountMinor, e.currency), e.stripeRefundId, e.status, e.reason, e.version);
    }

    private static RefundEntity toEntity(Refund r) {
        RefundEntity e = new RefundEntity();
        e.id = r.id();
        e.paymentId = r.paymentId();
        e.amountMinor = r.amount().amountMinor();
        e.currency = r.amount().currency();
        e.stripeRefundId = r.stripeRefundId();
        e.status = r.status();
        e.reason = r.reason();
        e.version = r.version();
        return e;
    }
}
