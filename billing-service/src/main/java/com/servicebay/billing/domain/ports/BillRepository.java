package com.servicebay.billing.domain.ports;

import com.servicebay.billing.domain.bill.Bill;
import java.util.List;
import java.util.Optional;

public interface BillRepository {

    /**
     * Inserts a new bill (id null) or updates an existing one, with its lines.
     * Throws {@link ConflictException} if another live bill exists for the
     * appointment, or the bill changed since it was read (version).
     */
    Bill save(Bill bill);

    Optional<Bill> findById(long id);

    /** Locks the row (SELECT ... FOR UPDATE) until the transaction ends. */
    Optional<Bill> findByIdForUpdate(long id);

    /** The appointment's bill that isn't VOID, if any. */
    Optional<Bill> findLiveByAppointment(long appointmentId);

    List<Bill> findByCustomer(long customerId);
}
