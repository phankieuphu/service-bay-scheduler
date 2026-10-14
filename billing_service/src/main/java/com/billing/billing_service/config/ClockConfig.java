package com.billing.billing_service.config;

import java.time.Clock;
import java.time.ZoneId;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class ClockConfig {

  // "Today" for warranty checks is the hub's calendar day, not the server's.
  @Bean
  Clock clock(@Value("${billing.time-zone:Asia/Ho_Chi_Minh}") String timeZone) {
    return Clock.system(ZoneId.of(timeZone));
  }
}
