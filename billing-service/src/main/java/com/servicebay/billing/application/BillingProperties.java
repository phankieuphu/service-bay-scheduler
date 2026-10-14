package com.servicebay.billing.application;

import java.math.BigDecimal;
import org.springframework.boot.context.properties.ConfigurationProperties;

/**
 * @param currency the ISO 4217 code bills are in when billing has to assume one (it must match scheduler's catalog)
 * @param warrantyFee the warranty/expiration fee in major units of the bill's currency (business-requirement §6;
 *     the amount is still open, §10 #6)
 * @param timeZone the hub day a service date is read in, for the warranty check
 * @param depositPolicy FIXED, PERCENT or FULL (§10 #5 is open, so it's configuration)
 * @param depositValue the FIXED amount (major units) or the PERCENT share
 */
@ConfigurationProperties(prefix = "billing")
public record BillingProperties(String currency, BigDecimal warrantyFee, String timeZone, String depositPolicy,
        BigDecimal depositValue) {
}
