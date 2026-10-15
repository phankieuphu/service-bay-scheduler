package com.servicebay.billing.application;

import java.util.Set;

/**
 * Who is making a request: identity-service's user id and role, and their
 * access token, forwarded to customer- and scheduler-service. A CUSTOMER's
 * customer id is looked up lazily, only by the use cases that need it.
 */
public record Caller(String userId, String role, String token) {

    private static final Set<String> STAFF = Set.of("TECHNICIAN", "MANAGER", "ADMIN");

    public boolean isStaff() {
        return STAFF.contains(role);
    }

    public boolean isCustomer() {
        return "CUSTOMER".equals(role);
    }
}
