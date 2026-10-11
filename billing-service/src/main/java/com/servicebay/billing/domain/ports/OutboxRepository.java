package com.servicebay.billing.domain.ports;

import com.servicebay.billing.domain.outbox.OutboxMessage;

public interface OutboxRepository {

    /** Call inside the transaction that made the change the message describes. */
    void append(OutboxMessage message);
}
