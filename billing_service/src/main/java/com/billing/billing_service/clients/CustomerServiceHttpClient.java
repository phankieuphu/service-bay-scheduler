package com.billing.billing_service.clients;

import java.time.Duration;
import java.util.Optional;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClient;

import com.billing.billing_service.ports.CustomerDirectory;

// GET {customer-service}/api/v1/customer/{id}; 404 means unknown or deleted.
@Component
public class CustomerServiceHttpClient implements CustomerDirectory {

  private final RestClient restClient;

  public CustomerServiceHttpClient(RestClient.Builder builder,
      @Value("${customer-service.url}") String baseUrl,
      @Value("${upstream.connect-timeout:2s}") Duration connectTimeout,
      @Value("${upstream.read-timeout:5s}") Duration readTimeout) {
    this.restClient = RestClients.build(builder, baseUrl, connectTimeout, readTimeout);
  }

  @Override
  public Optional<Customer> findCustomer(Long customerId) {
    CustomerBody body = RestClients.getOrNull("customer-service", "get customer " + customerId,
        () -> restClient.get().uri("/api/v1/customer/{id}", customerId).retrieve().body(CustomerBody.class));
    return Optional.ofNullable(body).map(b -> new Customer(b.id(), b.name(), b.status()));
  }

  record CustomerBody(Long id, String name, String status) {
  }
}
