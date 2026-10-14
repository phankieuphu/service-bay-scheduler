package com.servicebay.billing.adapters.kafka;

/** The record isn't a valid event of its topic: no retry will help, it goes to the DLT. */
class UnreadableEventException extends RuntimeException {

    UnreadableEventException(String message, Throwable cause) {
        super(message, cause);
    }
}
