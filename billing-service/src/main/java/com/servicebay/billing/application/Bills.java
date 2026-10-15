package com.servicebay.billing.application;

import com.servicebay.billing.domain.balance.Balance;
import com.servicebay.billing.domain.bill.Bill;
import com.servicebay.billing.domain.payment.Payment;
import com.servicebay.billing.domain.ports.BillRepository;
import com.servicebay.billing.domain.ports.PaymentRepository;
import com.servicebay.billing.domain.ports.RefundRepository;
import java.util.List;
import org.springframework.stereotype.Service;

/** Reading bills (#87). Customers see only their own (#90). */
@Service
public class Bills {

    /** Page size bounds for a customer's bill history. */
    static final int DEFAULT_LIMIT = 20;
    static final int MAX_LIMIT = 100;

    private final BillRepository bills;
    private final PaymentRepository payments;
    private final RefundRepository refunds;
    private final Access access;

    Bills(BillRepository bills, PaymentRepository payments, RefundRepository refunds, Access access) {
        this.bills = bills;
        this.payments = payments;
        this.refunds = refunds;
        this.access = access;
    }

    /** A bill with what's been paid toward it. */
    public record BillView(Bill bill, List<Payment> payments, Balance balance) {
    }

    /** One page, newest first; nextCursor is null on the last page. */
    public record Page(List<BillView> bills, Long nextCursor) {
    }

    public BillView get(long billId, Caller caller) {
        Bill bill = bills.findById(billId).orElseThrow(() -> new NotFoundException("bill not found"));
        access.requireCanRead(caller, bill.customerId(), "bill");
        return view(bill);
    }

    public BillView forAppointment(long appointmentId, Caller caller) {
        Bill bill = bills.findLiveByAppointment(appointmentId)
                .orElseThrow(() -> new NotFoundException("no bill for this appointment yet"));
        access.requireCanRead(caller, bill.customerId(), "bill");
        return view(bill);
    }

    public Page forCustomer(long customerId, Long cursor, Integer limit, Caller caller) {
        access.requireCanRead(caller, customerId, "customer");
        return page(customerId, cursor, limit);
    }

    public Page mine(Long cursor, Integer limit, Caller caller) {
        return page(access.customerIdOf(caller), cursor, limit);
    }

    private Page page(long customerId, Long cursor, Integer limit) {
        int size = limit == null || limit <= 0 ? DEFAULT_LIMIT : Math.min(limit, MAX_LIMIT);
        // One extra row tells whether another page exists.
        List<Bill> rows = bills.findByCustomer(customerId, cursor, size + 1);
        boolean more = rows.size() > size;
        List<Bill> pageRows = more ? rows.subList(0, size) : rows;
        Long next = more ? pageRows.get(pageRows.size() - 1).id() : null;
        return new Page(pageRows.stream().map(this::view).toList(), next);
    }

    private BillView view(Bill bill) {
        List<Payment> paid = payments.findByBill(bill.id());
        return new BillView(bill, paid, Balance.of(bill, paid, refunds.findByBill(bill.id())));
    }
}
