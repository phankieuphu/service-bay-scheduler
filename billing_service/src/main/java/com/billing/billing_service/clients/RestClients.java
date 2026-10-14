package com.billing.billing_service.clients;

import java.net.http.HttpClient;
import java.time.Duration;
import java.util.function.Supplier;

import org.springframework.http.client.JdkClientHttpRequestFactory;
import org.springframework.web.client.RestClient;
import org.springframework.web.client.RestClientException;
import org.springframework.web.client.RestClientResponseException;

import com.billing.billing_service.exceptions.UpstreamServiceException;

// Shared plumbing for billing's outbound HTTP clients.
final class RestClients {

  private RestClients() {
  }

  static RestClient build(RestClient.Builder builder, String baseUrl, Duration connectTimeout, Duration readTimeout) {
    JdkClientHttpRequestFactory requestFactory = new JdkClientHttpRequestFactory(
        HttpClient.newBuilder().connectTimeout(connectTimeout).build());
    requestFactory.setReadTimeout(readTimeout);
    return builder.clone().baseUrl(baseUrl).requestFactory(requestFactory).build();
  }

  // Runs a GET; returns null on 404 so callers can map it to Optional.empty().
  // Anything else that isn't a success becomes an UpstreamServiceException.
  static <T> T getOrNull(String service, String action, Supplier<T> request) {
    try {
      return request.get();
    } catch (RestClientResponseException e) {
      if (e.getStatusCode().value() == 404) {
        return null;
      }
      throw new UpstreamServiceException(service + " failed to " + action + " (" + e.getStatusCode().value()
          + "): " + e.getResponseBodyAsString(), e);
    } catch (RestClientException e) {
      throw new UpstreamServiceException(service + " unreachable while trying to " + action, e);
    }
  }
}
