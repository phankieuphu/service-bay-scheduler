package com.payment.payment_service.events;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.TimeUnit;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import com.payment.payment_service.entity.OutboxMessageEntity;
import com.payment.payment_service.repositories.OutboxRepository;

// Polls outbox_message and publishes unpublished rows to Kafka, then marks them
// published. Same contract as the Go services' kafka.OutboxRelay: it is the only
// code that publishes, and it stops at the first failure in a batch so per-key
// ordering is kept; the rest is retried on the next poll. Delivery is
// at-least-once, so consumers dedupe.
@Component
public class OutboxRelay {

  private static final Logger log = LoggerFactory.getLogger(OutboxRelay.class);

  private final OutboxRepository outboxRepository;
  private final KafkaTemplate<String, String> kafkaTemplate;
  private final int batchSize;

  public OutboxRelay(OutboxRepository outboxRepository, KafkaTemplate<String, String> kafkaTemplate,
      @Value("${outbox.batch-size:100}") int batchSize) {
    this.outboxRepository = outboxRepository;
    this.kafkaTemplate = kafkaTemplate;
    this.batchSize = batchSize;
  }

  @Scheduled(fixedDelayString = "${outbox.relay-interval-ms:2000}")
  public void relayOnce() {
    List<OutboxMessageEntity> messages;
    try {
      messages = outboxRepository.fetchUnpublished(batchSize);
    } catch (RuntimeException e) {
      log.error("outbox relay: failed to fetch unpublished messages", e);
      return;
    }

    List<Long> published = new ArrayList<>(messages.size());
    for (OutboxMessageEntity msg : messages) {
      try {
        kafkaTemplate.send(msg.getTopic(), msg.getMessageKey(), msg.getPayload()).get(10, TimeUnit.SECONDS);
      } catch (InterruptedException e) {
        Thread.currentThread().interrupt();
        break;
      } catch (Exception e) {
        log.error("outbox relay: failed to publish message {} to {}", msg.getId(), msg.getTopic(), e);
        break;
      }
      published.add(msg.getId());
    }

    if (published.isEmpty()) {
      return;
    }
    try {
      outboxRepository.markPublished(published);
    } catch (RuntimeException e) {
      log.error("outbox relay: failed to mark messages {} published", published, e);
    }
  }
}
