package com.servicebay.billing.adapters.http;

import com.servicebay.billing.adapters.http.Dtos.BillDto;
import com.servicebay.billing.adapters.http.Dtos.BillPageDto;
import com.servicebay.billing.adapters.http.Dtos.PaymentDto;
import com.servicebay.billing.adapters.http.Dtos.StartedDto;
import com.servicebay.billing.adapters.http.Dtos.TestConfirmRequest;
import com.servicebay.billing.application.Bills;
import com.servicebay.billing.application.Caller;
import com.servicebay.billing.application.Payments;
import org.springframework.http.HttpStatus;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

/**
 * The billing API (#86, #87, #88). Every route needs an identity-service
 * access token. A customer sees and pays only their own bills; someone
 * else's answers 404, not 403, so ids can't be probed (#90). Staff can read
 * every bill and re-sync a payment.
 */
@RestController
@RequestMapping("/api/v1")
class BillingController {

    private final Bills bills;
    private final Payments payments;

    BillingController(Bills bills, Payments payments) {
        this.bills = bills;
        this.payments = payments;
    }

    @GetMapping("/bills/{id}")
    BillDto getBill(@PathVariable long id, @AuthenticationPrincipal Jwt jwt) {
        return BillDto.of(bills.get(id, caller(jwt)));
    }

    @GetMapping("/bookings/{id}/bill")
    BillDto getBookingBill(@PathVariable long id, @AuthenticationPrincipal Jwt jwt) {
        return BillDto.of(bills.forAppointment(id, caller(jwt)));
    }

    @GetMapping("/customers/{id}/bills")
    BillPageDto customerBills(@PathVariable long id, @RequestParam(required = false) Long cursor,
            @RequestParam(required = false) Integer limit, @AuthenticationPrincipal Jwt jwt) {
        return BillPageDto.of(bills.forCustomer(id, cursor, limit, caller(jwt)));
    }

    @GetMapping("/me/bills")
    BillPageDto myBills(@RequestParam(required = false) Long cursor, @RequestParam(required = false) Integer limit,
            @AuthenticationPrincipal Jwt jwt) {
        return BillPageDto.of(bills.mine(cursor, limit, caller(jwt)));
    }

    /** Idempotency-Key is required: a retry with the same key returns the same payment. */
    @PostMapping("/bookings/{id}/deposit")
    @ResponseStatus(HttpStatus.CREATED)
    StartedDto deposit(@PathVariable long id, @RequestHeader(name = "Idempotency-Key", required = false) String key,
            @AuthenticationPrincipal Jwt jwt) {
        return StartedDto.of(payments.requestDeposit(id, caller(jwt), key));
    }

    /** Pays the bill's current balance; the client never sends an amount. */
    @PostMapping("/bills/{id}/payments")
    @ResponseStatus(HttpStatus.CREATED)
    StartedDto payBalance(@PathVariable long id, @RequestHeader(name = "Idempotency-Key", required = false) String key,
            @AuthenticationPrincipal Jwt jwt) {
        return StartedDto.of(payments.payBalance(id, caller(jwt), key));
    }

    /** What the return page polls: the webhook, not the redirect, decides success (#92). */
    @GetMapping("/payments/{id}")
    PaymentDto getPayment(@PathVariable long id, @AuthenticationPrincipal Jwt jwt) {
        return PaymentDto.of(payments.getPayment(id, caller(jwt)));
    }

    @PostMapping("/payments/{id}/sync")
    PaymentDto sync(@PathVariable long id, @AuthenticationPrincipal Jwt jwt) {
        return PaymentDto.of(payments.sync(id, caller(jwt)));
    }

    /** Stripe test-mode keys only (403 otherwise). */
    @PostMapping("/payments/{id}/test-confirm")
    PaymentDto testConfirm(@PathVariable long id, @RequestBody(required = false) TestConfirmRequest body,
            @AuthenticationPrincipal Jwt jwt) {
        return PaymentDto.of(payments.confirmTest(id, body == null ? null : body.paymentMethod(), caller(jwt)));
    }

    private static Caller caller(Jwt jwt) {
        return new Caller(jwt.getSubject(), jwt.getClaimAsString("role"), jwt.getTokenValue());
    }
}
