package kafka

import (
	"context"
	"errors"
	"customer-service/config"
	"customer-service/pkg/logger"

	"github.com/IBM/sarama"
)

type Producer struct {
	client   sarama.Client
	producer sarama.SyncProducer
}

func NewProducer(cfg config.Kafka) (*Producer, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Return.Errors = true
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	saramaCfg.Producer.Retry.Max = 3

	client, err := sarama.NewClient(cfg.Brokers, saramaCfg)
	if err != nil {
		return nil, err
	}
	p, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		client.Close()
		return nil, err
	}

	logger.Info("Kafka producer connected", "brokers", cfg.Brokers)
	return &Producer{client: client, producer: p}, nil
}

func (p *Producer) Publish(_ context.Context, topic, key string, payload []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(payload),
	}
	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return err
	}
	logger.Info("published", "topic", topic, "partition", partition, "offset", offset)
	return nil
}

// Ping asks the cluster for its current controller, a metadata round trip
// that fails if no broker is reachable. sarama takes no ctx, so callers
// enforce their own deadline.
func (p *Producer) Ping(_ context.Context) error {
	_, err := p.client.RefreshController()
	return err
}

// Close closes the producer, then the client it was built from (a producer
// created from a client doesn't close that client itself).
func (p *Producer) Close() error {
	return errors.Join(p.producer.Close(), p.client.Close())
}
