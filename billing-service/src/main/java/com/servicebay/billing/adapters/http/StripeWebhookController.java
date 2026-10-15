package com.servicebay.billing.adapters.http;

import com.servicebay.billing.application.Payments;
import com.servicebay.billing.domain.ports.PaymentProvider;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

/**
 * Stripe posts events here (#83). No JWT: the signature over the raw body is
 * the authentication, which is why the body is taken as a String and never
 * re-serialized. 200 handled or deliberately ignored; 400 bad signature; 5xx
 * anything else, which makes Stripe retry with backoff for up to three days.
 */
@RestController
class StripeWebhookController {

    private final PaymentProvider provider;
    private final Payments payments;

    StripeWebhookController(PaymentProvider provider, Payments payments) {
        this.provider = provider;
        this.payments = payments;
    }

    @PostMapping("/webhooks/stripe")
    ResponseEntity<Void> handle(@RequestBody String payload,
            @RequestHeader(name = "Stripe-Signature", required = false) String signature) {
        payments.handleWebhook(provider.parseWebhook(payload, signature));
        return ResponseEntity.ok().build();
    }
}
