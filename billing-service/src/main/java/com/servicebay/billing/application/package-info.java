/**
 * Use cases and Spring wiring. Each use case reads and writes through the
 * domain's ports and draws its own transaction boundaries, so a call to Stripe
 * or another service never runs while a database transaction or row lock is
 * held. The domain itself stays free of Spring.
 */
package com.servicebay.billing.application;
