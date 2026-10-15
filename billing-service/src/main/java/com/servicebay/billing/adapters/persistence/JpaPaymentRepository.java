package com.servicebay.billing.adapters.persistence;

import com.servicebay.billing.adapters.persistence.SpringDataRepositories.PaymentJpa;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.payment.Payment;
import com.servicebay.billing.domain.ports.ConflictException;
import com.servicebay.billing.domain.ports.PaymentRepository;
import java.util.List;
import java.util.Optional;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.orm.ObjectOptimisticLockingFailureException;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Component
class JpaPaymentRepository implements PaymentRepository {

    private final PaymentJpa payments;

    JpaPaymentRepository(PaymentJpa payments) {
        this.payments = payments;
    }

    @Override
    @Transactional
    public Payment save(Payment payment) {
        try {
            return toDomain(payments.saveAndFlush(toEntity(payment)));
        } catch (DataIntegrityViolationException e) {
            throw new ConflictException("payment conflicts with an existing one (idempotency key or PaymentIntent)", e);
        } catch (ObjectOptimisticLockingFailureException e) {
            throw new ConflictException("payment " + payment.id() + " was changed by someone else", e);
        }
    }

    @Override
    @Transactional(readOnly = true)
    public Optional<Payment> findById(long id) {
        return payments.findById(id).map(JpaPaymentRepository::toDomain);
    }

    @Override
    @Transactional(readOnly = true)
    public Optional<Payment> findByIdempotencyKey(String idempotencyKey) {
        return payments.findByIdempotencyKey(idempotencyKey).map(JpaPaymentRepository::toDomain);
    }

    @Override
    @Transactional(readOnly = true)
    public Optional<Payment> findByStripePaymentIntentId(String paymentIntentId) {
        return payments.findByStripePaymentIntentId(paymentIntentId).map(JpaPaymentRepository::toDomain);
    }

    @Override
    @Transactional(readOnly = true)
    public List<Payment> findByBill(long billId) {
        return payments.findByBillIdOrderById(billId).stream().map(JpaPaymentRepository::toDomain).toList();
    }

    private static Payment toDomain(PaymentEntity e) {
        return new Payment(e.id, e.billId, e.type, Money.of(e.amountMinor, e.currency), e.status,
                e.stripePaymentIntentId, e.idempotencyKey, e.failureCode, e.failureMessage, e.cardBrand, e.cardLast4,
                e.succeededAt, e.version);
    }

    private static String truncate(String value, int max) {
        return value == null || value.length() <= max ? value : value.substring(0, max);
    }

    private static PaymentEntity toEntity(Payment p) {
        PaymentEntity e = new PaymentEntity();
        e.id = p.id();
        e.billId = p.billId();
        e.type = p.type();
        e.amountMinor = p.amount().amountMinor();
        e.currency = p.amount().currency();
        e.status = p.status();
        e.stripePaymentIntentId = p.stripePaymentIntentId();
        e.idempotencyKey = p.idempotencyKey();
        e.failureCode = p.failureCode();
        e.failureMessage = truncate(p.failureMessage(), 500);
        e.cardBrand = p.cardBrand();
        e.cardLast4 = p.cardLast4();
        e.succeededAt = p.succeededAt();
        e.version = p.version();
        return e;
    }
}
