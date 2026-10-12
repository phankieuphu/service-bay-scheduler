package com.servicebay.billing.adapters.persistence;

import com.servicebay.billing.adapters.persistence.SpringDataRepositories.BillJpa;
import com.servicebay.billing.adapters.persistence.SpringDataRepositories.BillLineJpa;
import com.servicebay.billing.domain.bill.Bill;
import com.servicebay.billing.domain.bill.BillLine;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.ports.BillRepository;
import com.servicebay.billing.domain.ports.ConflictException;
import java.util.List;
import java.util.Optional;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.orm.ObjectOptimisticLockingFailureException;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Component
class JpaBillRepository implements BillRepository {

    private final BillJpa bills;
    private final BillLineJpa lines;

    JpaBillRepository(BillJpa bills, BillLineJpa lines) {
        this.bills = bills;
        this.lines = lines;
    }

    @Override
    @Transactional
    public Bill save(Bill bill) {
        BillEntity entity;
        try {
            entity = bills.saveAndFlush(toEntity(bill));
            // Lines are immutable once written: only new ones are inserted.
            for (BillLine line : bill.lines()) {
                if (line.id() == null) {
                    lines.save(toEntity(entity.id, line));
                }
            }
            lines.flush();
        } catch (DataIntegrityViolationException e) {
            throw new ConflictException("bill conflicts with an existing bill for appointment " + bill.appointmentId(), e);
        } catch (ObjectOptimisticLockingFailureException e) {
            throw new ConflictException("bill " + bill.id() + " was changed by someone else", e);
        }
        return toDomain(entity);
    }

    @Override
    @Transactional(readOnly = true)
    public Optional<Bill> findById(long id) {
        return bills.findById(id).map(this::toDomain);
    }

    @Override
    @Transactional
    public Optional<Bill> findByIdForUpdate(long id) {
        return bills.findByIdForUpdate(id).map(this::toDomain);
    }

    @Override
    @Transactional(readOnly = true)
    public Optional<Bill> findLiveByAppointment(long appointmentId) {
        return bills.findLiveByAppointment(appointmentId).map(this::toDomain);
    }

    @Override
    @Transactional(readOnly = true)
    public List<Bill> findByCustomer(long customerId) {
        return bills.findByCustomerIdOrderByCreatedAtDesc(customerId).stream().map(this::toDomain).toList();
    }

    private Bill toDomain(BillEntity e) {
        List<BillLine> billLines = lines.findByBillIdOrderById(e.id).stream()
                .map(l -> new BillLine(l.id, l.kind, l.sourceId, l.name, l.quantity, Money.of(l.unitPriceMinor, e.currency)))
                .toList();
        return new Bill(e.id, e.appointmentId, e.customerId, e.vehicleId, e.currency,
                Money.of(e.warrantyFeeMinor, e.currency), Money.of(e.serviceFeeMinor, e.currency),
                Money.of(e.materialCostMinor, e.currency), e.status, e.finalizedAt, billLines, e.version);
    }

    private static BillEntity toEntity(Bill b) {
        BillEntity e = new BillEntity();
        e.id = b.id();
        e.appointmentId = b.appointmentId();
        e.customerId = b.customerId();
        e.vehicleId = b.vehicleId();
        e.currency = b.currency();
        e.warrantyFeeMinor = b.warrantyFee().amountMinor();
        e.serviceFeeMinor = b.serviceFee().amountMinor();
        e.materialCostMinor = b.materialCost().amountMinor();
        e.status = b.status();
        e.finalizedAt = b.finalizedAt();
        e.version = b.version();
        return e;
    }

    private static BillLineEntity toEntity(long billId, BillLine l) {
        BillLineEntity e = new BillLineEntity();
        e.billId = billId;
        e.kind = l.kind();
        e.sourceId = l.sourceId();
        e.name = l.name();
        e.quantity = l.quantity();
        e.unitPriceMinor = l.unitPrice().amountMinor();
        return e;
    }
}
