package com.servicebay.billing.application;

import com.servicebay.billing.domain.ports.Directories.CustomerDirectory;
import org.springframework.stereotype.Component;

/**
 * Who may see or pay what (#90): staff see every bill; a customer only their
 * own. Staff aren't scoped to a dealership yet, because nothing links a staff
 * user to one.
 */
@Component
public class Access {

    private final CustomerDirectory customers;

    Access(CustomerDirectory customers) {
        this.customers = customers;
    }

    /** The caller's customer id; NotFound if they have no customer profile. */
    public long customerIdOf(Caller caller) {
        return customers.currentCustomerId(caller.token())
                .orElseThrow(() -> new NotFoundException("no customer profile is linked to this account"));
    }

    /** Throws NotFound unless the caller is staff or owns customerId's data. */
    public void requireCanRead(Caller caller, long customerId, String what) {
        if (caller.isStaff()) {
            return;
        }
        if (!caller.isCustomer() || customerIdOf(caller) != customerId) {
            throw new NotFoundException(what + " not found");
        }
    }

    /** Paying is the customer's own action: only the owner may start one. */
    public void requireOwner(Caller caller, long customerId, String what) {
        if (!caller.isCustomer() || customerIdOf(caller) != customerId) {
            throw new NotFoundException(what + " not found");
        }
    }
}
