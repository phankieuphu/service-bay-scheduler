package ports

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type IConsumer interface {
	Start(ctx context.Context)
	ProcessMessage(ctx context.Context, msg types.Message) error
	DeleteMessage(ctx context.Context, receiptHandle string) error
}

type SNSMessage struct {
	Type      string `json:"Type"`
	MessageId string `json:"MessageId"`
	TopicArn  string `json:"TopicArn"`
	Message   string `json:"Message"`
	Timestamp string `json:"Timestamp"`
}
