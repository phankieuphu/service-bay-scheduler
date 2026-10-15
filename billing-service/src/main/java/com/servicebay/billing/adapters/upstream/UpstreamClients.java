package com.servicebay.billing.adapters.upstream;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.servicebay.billing.adapters.security.ServiceTokenIssuer;
import com.servicebay.billing.domain.booking.AppointmentRef;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.ports.Directories.CustomerDirectory;
import com.servicebay.billing.domain.ports.Directories.SchedulerDirectory;
import com.servicebay.billing.domain.ports.Directories.UpstreamException;
import com.servicebay.billing.domain.ports.Directories.VehicleDirectory;
import java.math.BigDecimal;
import java.net.http.HttpClient;
import java.time.LocalDate;
import java.util.List;
import java.util.Optional;
import java.util.function.Supplier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.client.JdkClientHttpRequestFactory;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClient;
import org.springframework.web.client.RestClientException;
import org.springframework.web.client.RestClientResponseException;

/** HTTP adapters for the Go services billing reads from. */
final class UpstreamClients {

    private UpstreamClients() {
    }

    static RestClient client(RestClient.Builder builder, String baseUrl, UpstreamProperties props) {
        JdkClientHttpRequestFactory factory = new JdkClientHttpRequestFactory(
                HttpClient.newBuilder().connectTimeout(props.timeout()).build());
        factory.setReadTimeout(props.timeout());
        return builder.clone().baseUrl(baseUrl).requestFactory(factory).build();
    }

    /** Runs a GET; null on 404, UpstreamException for anything else that isn't a success. */
    static <T> T getOrNull(String service, String action, Supplier<T> call) {
        try {
            return call.get();
        } catch (RestClientResponseException e) {
            if (e.getStatusCode().value() == 404) {
                return null;
            }
            throw new UpstreamException(service + " failed to " + action + " (" + e.getStatusCode().value() + ")", e);
        } catch (RestClientException e) {
            throw new UpstreamException(service + " unreachable while trying to " + action, e);
        }
    }

    static String bearer(String token) {
        return "Bearer " + token;
    }

    /** GET /customer/me with the caller's token. */
    @Component
    static class CustomerServiceClient implements CustomerDirectory {

        private final RestClient rest;

        CustomerServiceClient(RestClient.Builder builder, UpstreamProperties props) {
            this.rest = client(builder, props.customerApiUrl(), props);
        }

        @Override
        public Optional<Long> currentCustomerId(String token) {
            Me me = getOrNull("customer-service", "read the caller's customer profile",
                    () -> rest.get().uri("/customer/me").header("Authorization", bearer(token)).retrieve().body(Me.class));
            return Optional.ofNullable(me).map(Me::id);
        }

        record Me(Long id) {
        }
    }

    /** GET /vehicle/:id/warranty?date= with billing's service token. */
    @Component
    static class VehicleServiceClient implements VehicleDirectory {

        private final RestClient rest;
        private final ServiceTokenIssuer tokens;

        VehicleServiceClient(RestClient.Builder builder, UpstreamProperties props, ServiceTokenIssuer tokens) {
            this.rest = client(builder, props.vehicleApiUrl(), props);
            this.tokens = tokens;
        }

        @Override
        public Warranty warranty(long vehicleId, LocalDate asOf) {
            WarrantyBody body = getOrNull("vehicle-service", "read vehicle " + vehicleId + "'s warranty",
                    () -> rest.get().uri("/vehicle/{id}/warranty?date={date}", vehicleId, asOf.toString())
                            .header("Authorization", bearer(tokens.token()))
                            .retrieve().body(WarrantyBody.class));
            if (body == null) {
                // A completed appointment names a vehicle that should exist; retrying
                // won't fix this, but assuming it's in warranty would be wrong too.
                throw new UpstreamException("vehicle-service has no vehicle " + vehicleId, null);
            }
            LocalDate end = body.warrantyEndDate() == null ? null : LocalDate.parse(body.warrantyEndDate().substring(0, 10));
            return new Warranty(body.inWarranty(), end);
        }

        record WarrantyBody(@JsonProperty("in_warranty") boolean inWarranty,
                @JsonProperty("warranty_end_date") String warrantyEndDate) {
        }
    }

    /** GET /me/appointments with the caller's token. */
    @Component
    static class SchedulerServiceClient implements SchedulerDirectory {

        private final RestClient rest;
        private final String currency;

        SchedulerServiceClient(RestClient.Builder builder, UpstreamProperties props,
                @Value("${billing.currency}") String currency) {
            this.rest = client(builder, props.schedulerApiUrl(), props);
            this.currency = currency;
        }

        @Override
        public Optional<AppointmentRef> findMyAppointment(String token, long appointmentId) {
            MyAppointments body = getOrNull("scheduler-service", "list the caller's appointments",
                    () -> rest.get().uri("/me/appointments").header("Authorization", bearer(token))
                            .retrieve().body(MyAppointments.class));
            if (body == null || body.appointments() == null) {
                return Optional.empty();
            }
            // The REST response has no currency; scheduler's events do. Both come
            // from the same catalog, whose currency billing is configured with.
            return body.appointments().stream()
                    .filter(a -> a.id() == appointmentId)
                    .findFirst()
                    .map(a -> new AppointmentRef(a.id(), a.customerId(), a.vehicleId(), a.hubId(),
                            Money.ofMajor(new BigDecimal(a.totalAmount()), currency), status(a.status())));
        }

        private static AppointmentRef.Status status(String scheduler) {
            return switch (scheduler) {
                case "CANCELLED", "NO_SHOW" -> AppointmentRef.Status.CANCELLED;
                case "COMPLETED" -> AppointmentRef.Status.COMPLETED;
                default -> AppointmentRef.Status.BOOKED;
            };
        }

        record MyAppointments(List<Appointment> appointments) {
        }

        record Appointment(long id, @JsonProperty("hub_id") long hubId, @JsonProperty("customer_id") long customerId,
                @JsonProperty("vehicle_id") long vehicleId, String status,
                @JsonProperty("total_amount") String totalAmount) {
        }
    }
}
