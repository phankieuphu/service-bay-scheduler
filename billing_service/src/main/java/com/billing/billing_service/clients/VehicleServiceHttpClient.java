package com.billing.billing_service.clients;

import java.time.Duration;
import java.time.LocalDate;
import java.util.List;
import java.util.Optional;
import java.util.Set;
import java.util.stream.Collectors;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClient;

import com.billing.billing_service.exceptions.NotFoundException;
import com.billing.billing_service.ports.VehicleDirectory;
import com.fasterxml.jackson.annotation.JsonProperty;

// vehicle-service reads used by checkout (vehicle-handler.go):
//   GET /api/v1/vehicle/{id}
//   GET /api/v1/customers/{id}/vehicles   (current ownerships only)
//   GET /api/v1/vehicle/{id}/warranty?date=YYYY-MM-DD
@Component
public class VehicleServiceHttpClient implements VehicleDirectory {

  private final RestClient restClient;

  public VehicleServiceHttpClient(RestClient.Builder builder,
      @Value("${vehicle-service.url}") String baseUrl,
      @Value("${upstream.connect-timeout:2s}") Duration connectTimeout,
      @Value("${upstream.read-timeout:5s}") Duration readTimeout) {
    this.restClient = RestClients.build(builder, baseUrl, connectTimeout, readTimeout);
  }

  @Override
  public Optional<Vehicle> findVehicle(Long vehicleId) {
    VehicleBody body = RestClients.getOrNull("vehicle-service", "get vehicle " + vehicleId,
        () -> restClient.get().uri("/api/v1/vehicle/{id}", vehicleId).retrieve().body(VehicleBody.class));
    return Optional.ofNullable(body).map(b -> new Vehicle(b.id(), b.vin(), b.licensePlate(), b.status()));
  }

  @Override
  public Set<Long> currentVehicleIds(Long customerId) {
    CustomerVehiclesBody body = RestClients.getOrNull("vehicle-service", "list vehicles of customer " + customerId,
        () -> restClient.get().uri("/api/v1/customers/{id}/vehicles", customerId).retrieve()
            .body(CustomerVehiclesBody.class));
    if (body == null || body.vehicles() == null) {
      return Set.of();
    }
    return body.vehicles().stream().map(VehicleBody::id).collect(Collectors.toUnmodifiableSet());
  }

  @Override
  public Warranty warranty(Long vehicleId, LocalDate asOf) {
    WarrantyBody body = RestClients.getOrNull("vehicle-service", "get warranty of vehicle " + vehicleId,
        () -> restClient.get().uri("/api/v1/vehicle/{id}/warranty?date={date}", vehicleId, asOf.toString())
            .retrieve().body(WarrantyBody.class));
    if (body == null) {
      throw new NotFoundException("vehicle " + vehicleId + " not found");
    }
    return new Warranty(body.inWarranty(), body.warrantyEndDate(), body.asOf());
  }

  record VehicleBody(Long id, String vin, @JsonProperty("license_plate") String licensePlate, String status) {
  }

  record CustomerVehiclesBody(List<VehicleBody> vehicles) {
  }

  record WarrantyBody(@JsonProperty("in_warranty") boolean inWarranty,
      @JsonProperty("warranty_end_date") String warrantyEndDate,
      @JsonProperty("as_of") String asOf) {
  }
}
