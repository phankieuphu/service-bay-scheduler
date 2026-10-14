package com.servicebay.billing.adapters.upstream;

import java.time.Duration;
import org.springframework.boot.context.properties.ConfigurationProperties;

/** Base URLs (ending in /api/v1) of the services billing calls, and a per-call timeout. */
@ConfigurationProperties(prefix = "upstream")
public record UpstreamProperties(String customerApiUrl, String vehicleApiUrl, String schedulerApiUrl,
        Duration timeout) {
}
