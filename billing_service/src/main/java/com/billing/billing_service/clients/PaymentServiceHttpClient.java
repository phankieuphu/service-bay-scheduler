package com.billing.billing_service.clients;

import java.math.BigDecimal;
import java.time.Duration;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClient;
import org.springframework.web.client.RestClientException;
import org.springframework.web.client.RestClientResponseException;

import com.billing.billing_service.exceptions.ConflictException;
import com.billing.billing_service.exceptions.PaymentGatewayException;
import com.billing.billing_service.ports.PaymentServiceClient;

@Component
public class PaymentServiceHttpClient implements PaymentServiceClient {

  private final RestClient restClient;

  public PaymentServiceHttpClient(RestClient.Builder builder,
      @Value("${payment-service.url}") String baseUrl,
      @Value("${payment-service.connect-timeout:2s}") Duration connectTimeout,
      @Value("${payment-service.read-timeout:15s}") Duration readTimeout) {
    this.restClient = RestClients.build(builder, baseUrl, connectTimeout, readTimeout);
  }

  @Override
  public CardCheckout createCardPayment(Long billId, Long billingPaymentId, Long customerId, BigDecimal amount,
      String currency) {
    CreateRequest body = new CreateRequest(billId, billingPaymentId, customerId, amount, currency);
    PaymentBody created = call("create card payment for billing payment " + billingPaymentId,
        () -> restClient.post()
            .uri("/api/v1/payments")
            .contentType(MediaType.APPLICATION_JSON)
            .body(body)
            .retrieve()
            .body(PaymentBody.class));
    if (created == null || created.id() == null) {
      throw new PaymentGatewayException("payment-service returned an empty checkout", null);
    }
    return new CardCheckout(created.id(), created.clientSecret());
  }

  @Override
  public void cancelCardPayment(Long externalPaymentId) {
    call("cancel payment " + externalPaymentId,
        () -> restClient.post()
            .uri("/api/v1/payments/{id}/cancel", externalPaymentId)
            .retrieve()
            .toBodilessEntity());
  }

  private static <T> T call(String action, java.util.function.Supplier<T> request) {
    try {
      return request.get();
    } catch (RestClientResponseException e) {
      String detail = e.getResponseBodyAsString();
      if (e.getStatusCode().value() == HttpStatus.CONFLICT.value()) {
        throw new ConflictException("payment-service refused to " + action + ": " + detail);
      }
      throw new PaymentGatewayException(
          "payment-service failed to " + action + " (" + e.getStatusCode().value() + "): " + detail, e);
    } catch (RestClientException e) {
      throw new PaymentGatewayException("payment-service unreachable while trying to " + action, e);
    }
  }

  record CreateRequest(Long billId, Long billingPaymentId, Long customerId, BigDecimal amount, String currency) {
  }

  record PaymentBody(Long id, String status, String clientSecret) {
  }
}
