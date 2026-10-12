package com.servicebay.billing.application;

import org.springframework.context.annotation.Configuration;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;

/** The Spring Data interfaces are nested in adapters.persistence.SpringDataRepositories. */
@Configuration
@EnableJpaRepositories(basePackages = "com.servicebay.billing.adapters.persistence", considerNestedRepositories = true)
class PersistenceConfig {
}
