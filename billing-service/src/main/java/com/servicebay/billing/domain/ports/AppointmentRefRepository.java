package com.servicebay.billing.domain.ports;

import com.servicebay.billing.domain.booking.AppointmentRef;
import java.util.Optional;

public interface AppointmentRefRepository {

    /** Inserts or replaces the snapshot for its appointment. */
    void upsert(AppointmentRef ref);

    Optional<AppointmentRef> findById(long appointmentId);
}
