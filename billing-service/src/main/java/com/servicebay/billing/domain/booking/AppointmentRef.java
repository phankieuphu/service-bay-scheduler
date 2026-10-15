package com.servicebay.billing.domain.booking;

import com.servicebay.billing.domain.money.Money;
import java.util.Objects;

/**
 * Billing's copy of a booking (architecture-design §5b): who it's for, which
 * vehicle, and the booked services' estimate that a deposit is a share of.
 * Filled from scheduler's AppointmentBooked/AppointmentCancelled events.
 */
public record AppointmentRef(
        long appointmentId,
        long customerId,
        long vehicleId,
        long dealershipId,
        Money estimate,
        Status status) {

    public enum Status { BOOKED, CANCELLED, COMPLETED }

    public AppointmentRef {
        Objects.requireNonNull(estimate, "estimate");
        Objects.requireNonNull(status, "status");
        if (estimate.isNegative()) {
            throw new IllegalArgumentException("estimate must not be negative");
        }
    }

    public String currency() {
        return estimate.currency();
    }

    public AppointmentRef withStatus(Status next) {
        return new AppointmentRef(appointmentId, customerId, vehicleId, dealershipId, estimate, next);
    }
}
