package com.servicebay.billing.application;

import com.servicebay.billing.domain.deposit.DepositPolicy;
import java.time.Clock;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.EnableScheduling;

@Configuration
@EnableScheduling
class BillingConfig {

    @Bean
    Clock clock() {
        return Clock.systemUTC();
    }

    /** Fails startup on a bad DEPOSIT_POLICY/DEPOSIT_VALUE rather than at the first deposit. */
    @Bean
    DepositPolicy depositPolicy(BillingProperties properties) {
        return DepositPolicy.parse(properties.depositPolicy(), properties.depositValue());
    }
}
