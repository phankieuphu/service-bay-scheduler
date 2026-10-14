package com.billing.billing_service.controllers;

import java.math.BigDecimal;
import java.util.List;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import com.billing.billing_service.controllers.dto.BillLineResponse;
import com.billing.billing_service.controllers.dto.BillResponse;
import com.billing.billing_service.controllers.dto.CheckoutRequest;
import com.billing.billing_service.controllers.dto.CheckoutResponse;
import com.billing.billing_service.controllers.dto.CreateBillRequest;
import com.billing.billing_service.controllers.dto.PaymentRequest;
import com.billing.billing_service.controllers.dto.PaymentResponse;
import com.billing.billing_service.ports.BillCheckoutService;
import com.billing.billing_service.ports.BillingService;
import com.billing.billing_service.ports.BillingService.RecordedPayment;

@RestController
@RequestMapping("/api/v1/bills")
public class BillingServiceControler {
  private final BillingService billingService;

  private final BillCheckoutService checkoutService;

  public BillingServiceControler(BillingService billingService, BillCheckoutService checkoutService) {
    this.billingService = billingService;
    this.checkoutService = checkoutService;
  }

  // Service-completion workflow: verifies customer, vehicle and ownership with
  // their services, applies the warranty-fee rule, and issues an itemized bill.
  @PostMapping("/checkout")
  @ResponseStatus(HttpStatus.CREATED)
  public CheckoutResponse checkout(@RequestBody CheckoutRequest req) {
    if (req.appointmentId() == null || req.customerId() == null || req.vehicleId() == null) {
      throw new IllegalArgumentException("appointmentId, customerId and vehicleId are required");
    }
    if (req.services() == null || req.services().isEmpty()) {
      throw new IllegalArgumentException("services must list at least one service");
    }
    List<BillCheckoutService.Line> services = toLines("services", req.services());
    List<BillCheckoutService.Line> materials = req.materials() == null ? List.of() : toLines("materials", req.materials());

    return CheckoutResponse.from(checkoutService.checkout(new BillCheckoutService.CheckoutCommand(
        req.appointmentId(), req.customerId(), req.vehicleId(), req.serviceDate(), services, materials)));
  }

  @GetMapping("/{id}/lines")
  public List<BillLineResponse> getLines(@PathVariable Long id) {
    return checkoutService.getLines(id).stream().map(BillLineResponse::from).toList();
  }

  @PostMapping
  @ResponseStatus(HttpStatus.CREATED)
  public BillResponse createBill(@RequestBody CreateBillRequest req) {
    if (req.appointmentId() == null || req.customerId() == null || req.vehicleId() == null) {
      throw new IllegalArgumentException("appointmentId, customerId and vehicleId are required");
    }
    requireNonNegative("warrantyFee", req.warrantyFee());
    requireNonNegative("serviceFee", req.serviceFee());
    requireNonNegative("materialCost", req.materialCost());

    return BillResponse.from(billingService.createBill(req.appointmentId(), req.customerId(), req.vehicleId(),
        req.warrantyFee(), req.serviceFee(), req.materialCost()));
  }

  @GetMapping("/{id}")
  public BillResponse getBill(@PathVariable Long id) {
    return BillResponse.from(billingService.getBillingEntityById(id));
  }

  @GetMapping
  public List<BillResponse> getBillsByCustomer(@RequestParam Long customerId) {
    return billingService.getBillsByCustomer(customerId).stream().map(BillResponse::from).toList();
  }

  @PostMapping("/{id}/payments")
  @ResponseStatus(HttpStatus.CREATED)
  public PaymentResponse recordPayment(@PathVariable Long id, @RequestBody PaymentRequest req) {
    if (req.type() == null || req.method() == null) {
      throw new IllegalArgumentException("type and method are required");
    }
    if (req.amount() == null || req.amount().signum() <= 0) {
      throw new IllegalArgumentException("amount must be greater than 0");
    }
    requireCents("amount", req.amount());

    RecordedPayment recorded = billingService.recordPayment(id, req.type(), req.method(), req.amount());
    return PaymentResponse.from(recorded.payment(), recorded.clientSecret());
  }

  @GetMapping("/{id}/payments")
  public List<PaymentResponse> getPayments(@PathVariable Long id) {
    return billingService.getPayments(id).stream().map(PaymentResponse::from).toList();
  }

  // Abandons a CARD payment that hasn't been paid yet, releasing its amount.
  @PostMapping("/{id}/payments/{paymentId}/cancel")
  public PaymentResponse cancelCardPayment(@PathVariable Long id, @PathVariable Long paymentId) {
    return PaymentResponse.from(billingService.cancelCardPayment(id, paymentId));
  }

  @PostMapping("/{id}/void")
  public BillResponse voidBill(@PathVariable Long id) {
    return BillResponse.from(billingService.voidBill(id));
  }

  private static void requireNonNegative(String field, BigDecimal value) {
    if (value != null && value.signum() < 0) {
      throw new IllegalArgumentException(field + " must not be negative");
    }
    requireCents(field, value);
  }

  private static List<BillCheckoutService.Line> toLines(String field, List<CheckoutRequest.LineRequest> lines) {
    return lines.stream().map(l -> {
      if (l == null || l.description() == null || l.description().isBlank()) {
        throw new IllegalArgumentException(field + ": every line needs a description");
      }
      if (l.description().length() > 255) {
        throw new IllegalArgumentException(field + ": description must be at most 255 characters");
      }
      int quantity = l.quantity() == null ? 1 : l.quantity();
      if (quantity < 1) {
        throw new IllegalArgumentException(field + ": quantity must be at least 1");
      }
      if (l.unitPrice() == null) {
        throw new IllegalArgumentException(field + ": unitPrice is required");
      }
      requireNonNegative(field + ".unitPrice", l.unitPrice());
      return new BillCheckoutService.Line(l.description(), quantity, l.unitPrice());
    }).toList();
  }

  // Columns are numeric(12,2); reject rather than let Postgres round what is charged.
  private static void requireCents(String field, BigDecimal value) {
    if (value != null && value.stripTrailingZeros().scale() > 2) {
      throw new IllegalArgumentException(field + " must have at most 2 decimal places");
    }
  }
}
