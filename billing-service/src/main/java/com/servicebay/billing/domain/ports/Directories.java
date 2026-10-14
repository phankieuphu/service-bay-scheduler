package com.servicebay.billing.domain.ports;

import com.servicebay.billing.domain.booking.AppointmentRef;
import java.time.LocalDate;
import java.util.Optional;

/**
 * Billing's synchronous views of other services. Each throws
 * {@link UpstreamException} when the service can't answer, so a Kafka handler
 * retries instead of assuming anything (#85: a failed warranty lookup must
 * never be read as "in warranty").
 */
public final class Directories {

    private Directories() {
    }

    /** customer-service, called with the caller's own token. */
    public interface CustomerDirectory {

        /** The caller's customer id (GET /customer/me); empty if they have no customer profile. */
        Optional<Long> currentCustomerId(String bearerToken);
    }

    /** vehicle-service, called with billing's own service token. */
    public interface VehicleDirectory {

        Warranty warranty(long vehicleId, LocalDate asOf);

        /** @param endDate null when the vehicle has no warranty on record */
        record Warranty(boolean inWarranty, LocalDate endDate) {
        }
    }

    /** scheduler-service, called with the caller's own token. */
    public interface SchedulerDirectory {

        /**
         * One of the caller's own appointments (GET /me/appointments), for a
         * deposit request that arrives before the AppointmentBooked event.
         */
        Optional<AppointmentRef> findMyAppointment(String bearerToken, long appointmentId);
    }

    public static class UpstreamException extends RuntimeException {

        public UpstreamException(String message, Throwable cause) {
            super(message, cause);
        }
    }
}
