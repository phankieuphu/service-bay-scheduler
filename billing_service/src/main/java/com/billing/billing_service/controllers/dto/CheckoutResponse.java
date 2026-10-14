package com.billing.billing_service.controllers.dto;

import java.util.List;

import com.billing.billing_service.ports.BillCheckoutService.Checkout;

public record CheckoutResponse(BillResponse bill, List<BillLineResponse> lines, WarrantyResponse warranty) {

  // What the warranty-fee decision was based on.
  public record WarrantyResponse(boolean inWarranty, String warrantyEndDate, String asOf, boolean feeCharged) {
  }

  public static CheckoutResponse from(Checkout checkout) {
    return new CheckoutResponse(BillResponse.from(checkout.bill()),
        checkout.lines().stream().map(BillLineResponse::from).toList(),
        new WarrantyResponse(checkout.warranty().inWarranty(), checkout.warranty().warrantyEndDate(),
            checkout.warranty().asOf(), checkout.bill().getWarrantyFee().signum() > 0));
  }
}
