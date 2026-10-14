package com.billing.billing_service.services;

import java.math.BigDecimal;
import java.time.Clock;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionOperations;

import com.billing.billing_service.entity.BillLineEntity;
import com.billing.billing_service.entity.BillLineKind;
import com.billing.billing_service.entity.BillingEntity;
import com.billing.billing_service.exceptions.ConflictException;
import com.billing.billing_service.exceptions.NotFoundException;
import com.billing.billing_service.ports.BillCheckoutService;
import com.billing.billing_service.ports.CustomerDirectory;
import com.billing.billing_service.ports.VehicleDirectory;
import com.billing.billing_service.ports.VehicleDirectory.Warranty;
import com.billing.billing_service.repositories.BillLineRepository;
import com.billing.billing_service.repositories.BillingRepository;

// Remote checks run first, with no transaction open; the bill and its lines are
// then written in one short transaction. The checks are advisory in the sense of
// architecture-design.md §5a: ids are validated at write time, never re-validated.
@Service
public class BillCheckoutServiceImpl implements BillCheckoutService {

  private static final String ACTIVE = "ACTIVE";

  private final BillingRepository billingRepository;
  private final BillLineRepository billLineRepository;
  private final CustomerDirectory customers;
  private final VehicleDirectory vehicles;
  private final TransactionOperations tx;
  private final BigDecimal warrantyFee;
  private final Clock clock;

  public BillCheckoutServiceImpl(BillingRepository billingRepository, BillLineRepository billLineRepository,
      CustomerDirectory customers, VehicleDirectory vehicles, TransactionOperations tx,
      @Value("${billing.warranty-fee}") BigDecimal warrantyFee, Clock clock) {
    this.billingRepository = billingRepository;
    this.billLineRepository = billLineRepository;
    this.customers = customers;
    this.vehicles = vehicles;
    this.tx = tx;
    this.warrantyFee = warrantyFee;
    this.clock = clock;
  }

  @Override
  public Checkout checkout(CheckoutCommand cmd) {
    // Cheap local check first so a retried checkout doesn't fan out to two services.
    if (billingRepository.existsByAppointmentId(cmd.appointmentId())) {
      throw new ConflictException("bill already exists for appointment " + cmd.appointmentId());
    }

    CustomerDirectory.Customer customer = customers.findCustomer(cmd.customerId())
        .orElseThrow(() -> new NotFoundException("customer " + cmd.customerId() + " not found"));
    if (!ACTIVE.equals(customer.status())) {
      throw new ConflictException("customer " + customer.id() + " is " + customer.status());
    }

    VehicleDirectory.Vehicle vehicle = vehicles.findVehicle(cmd.vehicleId())
        .orElseThrow(() -> new NotFoundException("vehicle " + cmd.vehicleId() + " not found"));
    if (!ACTIVE.equals(vehicle.status())) {
      throw new ConflictException("vehicle " + vehicle.id() + " is " + vehicle.status());
    }
    if (!vehicles.currentVehicleIds(customer.id()).contains(vehicle.id())) {
      throw new ConflictException("vehicle " + vehicle.id() + " is not currently owned by customer " + customer.id());
    }

    LocalDate serviceDate = cmd.serviceDate() != null ? cmd.serviceDate() : LocalDate.now(clock);
    Warranty warranty = vehicles.warranty(vehicle.id(), serviceDate);

    return tx.execute(status -> {
      BigDecimal fee = warranty.inWarranty() ? BigDecimal.ZERO : warrantyFee;
      BillingEntity bill = billingRepository.save(new BillingEntity(cmd.appointmentId(), customer.id(),
          vehicle.id(), fee, total(cmd.services()), total(cmd.materials())));

      List<BillLineEntity> lines = new ArrayList<>();
      if (fee.signum() > 0) {
        String reason = warranty.warrantyEndDate() == null
            ? "no warranty on record"
            : "warranty ended " + warranty.warrantyEndDate().substring(0, Math.min(10, warranty.warrantyEndDate().length()));
        lines.add(new BillLineEntity(bill.getId(), BillLineKind.WARRANTY_FEE, "Warranty/expiration fee (" + reason + ")",
            1, fee));
      }
      cmd.services().forEach(l -> lines.add(line(bill, BillLineKind.SERVICE, l)));
      cmd.materials().forEach(l -> lines.add(line(bill, BillLineKind.MATERIAL, l)));
      return new Checkout(bill, billLineRepository.saveAll(lines), warranty);
    });
  }

  @Override
  public List<BillLineEntity> getLines(Long billId) {
    if (!billingRepository.existsById(billId)) {
      throw new NotFoundException("bill " + billId + " not found");
    }
    return billLineRepository.findByBillIdOrderByIdAsc(billId);
  }

  private static BillLineEntity line(BillingEntity bill, BillLineKind kind, Line l) {
    return new BillLineEntity(bill.getId(), kind, l.description().trim(), l.quantity(), l.unitPrice());
  }

  private static BigDecimal total(List<Line> lines) {
    return lines.stream()
        .map(l -> l.unitPrice().multiply(BigDecimal.valueOf(l.quantity())))
        .reduce(BigDecimal.ZERO, BigDecimal::add);
  }
}
