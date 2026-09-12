package consumer

import (
	"context"
	internal_config "customer-service/config"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func NewSQSClient(internal_config internal_config.Config, ctx context.Context) (*sqs.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(internal_config.Region))
	if err != nil {
		log.Fatal("❌ load aws config failed:", err)
		return nil, err
	}

	return sqs.NewFromConfig(cfg), nil
}
