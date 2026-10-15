package com.servicebay.billing.adapters.kafka;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.TimeUnit;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import org.springframework.transaction.support.TransactionOperations;

/**
 * Publishes outbox_message rows to Kafka and marks them published (#82).
 *
 * <p>Each pass runs in a transaction holding a Postgres advisory lock, so with
 * several replicas only one publishes at a time: no duplicates from two
 * relays reading the same rows, and no reordering of one key's events (the Go
 * services' #68). A pass stops at the first failed publish and leaves the rest
 * for the next one, which keeps per-key order.
 */
@Component
class OutboxRelay {

    private static final Logger log = LoggerFactory.getLogger(OutboxRelay.class);

    /** An arbitrary constant naming billing's relay lock; only billing_db uses it. */
    private static final long RELAY_LOCK = 0x6269_6c6c_6f75_7478L; // "billoutx"

    private final JdbcClient jdbc;
    private final KafkaTemplate<String, String> kafka;
    private final TransactionOperations tx;
    private final int batchSize;
    private final int retentionDays;

    OutboxRelay(JdbcClient jdbc, KafkaTemplate<String, String> kafka, TransactionOperations tx,
            @Value("${outbox.batch-size:100}") int batchSize, @Value("${outbox.retention-days:7}") int retentionDays) {
        this.jdbc = jdbc;
        this.kafka = kafka;
        this.tx = tx;
        this.batchSize = batchSize;
        this.retentionDays = retentionDays;
    }

    record Row(long id, String topic, String messageKey, String payload) {
    }

    @Scheduled(fixedDelayString = "${outbox.relay-interval:2s}")
    void relay() {
        try {
            tx.executeWithoutResult(s -> relayOnce());
        } catch (RuntimeException e) {
            log.error("outbox relay pass failed", e);
        }
    }

    int relayOnce() {
        Boolean locked = jdbc.sql("select pg_try_advisory_xact_lock(:key)").param("key", RELAY_LOCK)
                .query(Boolean.class).single();
        if (!Boolean.TRUE.equals(locked)) {
            return 0; // another replica's relay is mid-pass
        }
        List<Row> rows = jdbc.sql("""
                select id, topic, message_key, payload::text as payload from outbox_message
                where published_at is null order by id limit :limit""")
                .param("limit", batchSize)
                .query(Row.class).list();

        List<Long> published = new ArrayList<>(rows.size());
        for (Row row : rows) {
            try {
                kafka.send(row.topic(), row.messageKey(), row.payload()).get(10, TimeUnit.SECONDS);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                break;
            } catch (Exception e) {
                log.error("outbox relay: failed to publish message {} to {}; retrying next pass", row.id(), row.topic(), e);
                break;
            }
            published.add(row.id());
        }
        if (!published.isEmpty()) {
            jdbc.sql("update outbox_message set published_at = now() where id in (:ids)").param("ids", published).update();
        }
        return published.size();
    }

    /** Published rows are only an audit trail; keep a week of them. */
    @Scheduled(cron = "${outbox.cleanup-cron:0 17 3 * * *}")
    void cleanup() {
        int deleted = jdbc.sql("delete from outbox_message where published_at < now() - make_interval(days => :days)")
                .param("days", retentionDays).update();
        if (deleted > 0) {
            log.info("outbox cleanup: deleted {} published rows older than {} days", deleted, retentionDays);
        }
    }
}
