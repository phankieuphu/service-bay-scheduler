package com.servicebay.billing.application;

/**
 * Unknown, or not the caller's: a customer asking for someone else's bill
 * gets the same 404 as for one that doesn't exist, so ids can't be probed (#90).
 */
public class NotFoundException extends RuntimeException {

    public NotFoundException(String message) {
        super(message);
    }
}
