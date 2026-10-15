package com.servicebay.billing.adapters.kafka;

import org.apache.kafka.common.TopicPartition;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.listener.DeadLetterPublishingRecoverer;
import org.springframework.kafka.listener.DefaultErrorHandler;
import org.springframework.kafka.support.ExponentialBackOffWithMaxRetries;

/**
 * At-least-once consumption without blocking a partition forever (#82): a
 * failing record is retried with exponential backoff, then published to
 * {@code <topic>.DLT}. A record that can't be parsed goes straight there,
 * since retrying won't change it.
 */
@Configuration
class KafkaConfig {

    private static final Logger log = LoggerFactory.getLogger(KafkaConfig.class);

    @Bean
    DefaultErrorHandler kafkaErrorHandler(KafkaTemplate<String, String> template) {
        DeadLetterPublishingRecoverer recoverer = new DeadLetterPublishingRecoverer(template,
                // Partition -1: let the producer pick; the DLT needn't mirror the source's partitions.
                (record, e) -> {
                    log.error("sending {}-{}@{} to {}.DLT: {}", record.topic(), record.partition(), record.offset(),
                            record.topic(), e.getMessage());
                    return new TopicPartition(record.topic() + ".DLT", -1);
                });
        // 1s, 2s, 4s, 8s, 16s, then 30s: about two minutes before giving up.
        ExponentialBackOffWithMaxRetries backOff = new ExponentialBackOffWithMaxRetries(6);
        backOff.setInitialInterval(1_000);
        backOff.setMultiplier(2);
        backOff.setMaxInterval(30_000);
        DefaultErrorHandler handler = new DefaultErrorHandler(recoverer, backOff);
        handler.addNotRetryableExceptions(UnreadableEventException.class);
        return handler;
    }
}
