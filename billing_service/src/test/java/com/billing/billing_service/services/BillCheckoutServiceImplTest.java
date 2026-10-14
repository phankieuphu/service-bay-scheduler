package com.billing.billing_service.services;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;

import java.math.BigDecimal;
import java.time.Clock;
import java.time.Instant;
import java.time.LocalDate;
import java.time.ZoneId;
import java.util.List;
import java.util.Optional;
import java.util.Set;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.test.util.ReflectionTestUtils;
import org.springframework.transaction.support.TransactionOperations;

import com.billing.billing_service.entity.BillLineEntity;
import com.billing.billing_service.entity.BillLineKind;
import com.billing.billing_service.entity.BillingEntity;
import com.billing.billing_service.exceptions.ConflictException;
import com.billing.billing_service.exceptions.NotFoundException;
import com.billing.billing_service.ports.BillCheckoutService.Checkout;
import com.billing.billing_service.ports.BillCheckoutService.CheckoutCommand;
import com.billing.billing_service.ports.BillCheckoutService.Line;
import com.billing.billing_service.ports.CustomerDirectory;
import com.billing.billing_service.ports.CustomerDirectory.Customer;
import com.billing.billing_service.ports.VehicleDirectory;
import com.billing.billing_service.ports.VehicleDirectory.Vehicle;
import com.billing.billing_service.ports.VehicleDirectory.Warranty;
import com.billing.billing_service.repositories.BillLineRepository;
import com.billing.billing_service.repositories.BillingRepository;

class BillCheckoutServiceImplTest {

  // 2026-10-15 in Asia/Ho_Chi_Minh.
  private static final Clock CLOCK = Clock.fixed(Instant.parse("2026-10-15T03:00:00Z"), ZoneId.of("Asia/Ho_Chi_Minh"));

  private BillingRepository billingRepository;
  private BillLineRepository billLineRepository;
  private CustomerDirectory customers;
  private VehicleDirectory vehicles;
  private BillCheckoutServiceImpl service;

  @BeforeEach
  void setUp() {
    billingRepository = mock(BillingRepository.class);
    billLineRepository = mock(BillLineRepository.class);
    customers = mock(CustomerDirectory.class);
    vehicles = mock(VehicleDirectory.class);
    service = new BillCheckoutServiceImpl(billingRepository, billLineRepository, customers, vehicles,
        TransactionOperations.withoutTransaction(), new BigDecimal("50.00"), CLOCK);

    when(billingRepository.save(any(BillingEntity.class))).thenAnswer(inv -> {
      BillingEntity b = inv.getArgument(0);
      ReflectionTestUtils.setField(b, "id", 1L);
      return b;
    });
    when(billLineRepository.saveAll(anyList())).thenAnswer(inv -> inv.getArgument(0));

    when(customers.findCustomer(685L)).thenReturn(Optional.of(new Customer(685L, "Rashawn", "ACTIVE")));
    when(vehicles.findVehicle(1L)).thenReturn(Optional.of(new Vehicle(1L, "G2LY3GEJ0PUN50045", "MW00YEH", "ACTIVE")));
    when(vehicles.currentVehicleIds(685L)).thenReturn(Set.of(1L));
  }

  private static CheckoutCommand command(LocalDate serviceDate) {
    return new CheckoutCommand(500L, 685L, 1L, serviceDate,
        List.of(new Line("Oil change", 1, new BigDecimal("30.00")), new Line("Tire rotation", 2, new BigDecimal("15.00"))),
        List.of(new Line("Oil filter", 1, new BigDecimal("12.50")), new Line("Engine oil 1L", 4, new BigDecimal("8.00"))));
  }

  @Test
  void noWarrantyChargesTheFeeAndItemizesEverything() {
    when(vehicles.warranty(1L, LocalDate.of(2026, 10, 15))).thenReturn(new Warranty(false, null, "2026-10-15"));

    Checkout checkout = service.checkout(command(null));

    BillingEntity bill = checkout.bill();
    assertEquals(0, bill.getWarrantyFee().compareTo(new BigDecimal("50.00")));
    assertEquals(0, bill.getServiceFee().compareTo(new BigDecimal("60.00")));   // 30 + 2 x 15
    assertEquals(0, bill.getMaterialCost().compareTo(new BigDecimal("44.50"))); // 12.50 + 4 x 8
    assertEquals(0, bill.getTotalAmount().compareTo(new BigDecimal("154.50")));

    List<BillLineEntity> lines = checkout.lines();
    assertEquals(5, lines.size());
    assertEquals(BillLineKind.WARRANTY_FEE, lines.get(0).getKind());
    assertTrue(lines.get(0).getDescription().contains("no warranty on record"));
    assertEquals(0, lines.get(4).getAmount().compareTo(new BigDecimal("32.00")));
  }

  @Test
  void activeWarrantyWaivesTheFee() {
    when(vehicles.warranty(1L, LocalDate.of(2026, 10, 1)))
        .thenReturn(new Warranty(true, "2029-01-12T00:00:00Z", "2026-10-01"));

    Checkout checkout = service.checkout(command(LocalDate.of(2026, 10, 1)));

    assertEquals(0, checkout.bill().getWarrantyFee().signum());
    assertTrue(checkout.lines().stream().noneMatch(l -> l.getKind() == BillLineKind.WARRANTY_FEE));
  }

  @Test
  void expiredWarrantySaysWhenItEnded() {
    when(vehicles.warranty(anyLong(), any())).thenReturn(new Warranty(false, "2026-09-27T00:00:00Z", "2026-10-15"));

    Checkout checkout = service.checkout(command(null));

    assertEquals("Warranty/expiration fee (warranty ended 2026-09-27)", checkout.lines().get(0).getDescription());
  }

  @Test
  void duplicateAppointmentIsRejectedBeforeAnyRemoteCall() {
    when(billingRepository.existsByAppointmentId(500L)).thenReturn(true);

    assertThrows(ConflictException.class, () -> service.checkout(command(null)));
    verifyNoInteractions(customers, vehicles);
  }

  @Test
  void unknownCustomerIsNotFound() {
    when(customers.findCustomer(685L)).thenReturn(Optional.empty());

    assertThrows(NotFoundException.class, () -> service.checkout(command(null)));
    verify(billingRepository, never()).save(any());
  }

  @Test
  void inactiveCustomerIsConflict() {
    when(customers.findCustomer(685L)).thenReturn(Optional.of(new Customer(685L, "Rashawn", "BANNED")));

    assertThrows(ConflictException.class, () -> service.checkout(command(null)));
  }

  @Test
  void soldVehicleIsConflict() {
    when(vehicles.findVehicle(1L)).thenReturn(Optional.of(new Vehicle(1L, "VIN", null, "SOLD")));

    assertThrows(ConflictException.class, () -> service.checkout(command(null)));
  }

  @Test
  void vehicleOwnedBySomeoneElseIsConflict() {
    when(vehicles.currentVehicleIds(685L)).thenReturn(Set.of(7L));

    ConflictException e = assertThrows(ConflictException.class, () -> service.checkout(command(null)));
    assertEquals("vehicle 1 is not currently owned by customer 685", e.getMessage());
    verify(billingRepository, never()).save(any());
  }
}
