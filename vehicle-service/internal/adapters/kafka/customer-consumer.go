package kafka

import (
	"context"
	"vehicle-service/config"
	"vehicle-service/pkg/logger"

	"github.com/IBM/sarama"
)

// MessageHandler handles one message. topic says which subscription it
// came from, since one consumer group can subscribe to several.
type MessageHandler func(ctx context.Context, topic string, key, value []byte) error

type Consumer struct {
	group   sarama.ConsumerGroup
	topics  []string
	handler MessageHandler
}

// NewConsumer joins cfg.ConsumerGroup subscribed to topics. None of them may
// be one this service produces to, or it would consume its own output.
func NewConsumer(cfg config.Kafka, topics []string, handler MessageHandler) (*Consumer, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	saramaCfg.Version = sarama.V2_1_0_0

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroup, saramaCfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Kafka consumer connected", "group", cfg.ConsumerGroup, "topics", topics, "brokers", cfg.Brokers)
	return &Consumer{group: group, topics: topics, handler: handler}, nil
}

func (c *Consumer) Start(ctx context.Context) {
	h := &consumerGroupHandler{handler: c.handler}
	for {
		if err := c.group.Consume(ctx, c.topics, h); err != nil {
			logger.Error("kafka consumer error", "error", err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (c *Consumer) Close() error {
	return c.group.Close()
}

type consumerGroupHandler struct {
	handler MessageHandler
}

func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.handler(session.Context(), msg.Topic, msg.Key, msg.Value); err != nil {
			logger.Error("message handling error", "error", err)
		} else {
			session.MarkMessage(msg, "")
		}
	}
	return nil
}
