package consumer

import "github.com/aws/aws-sdk-go-v2/service/sqs"

type QueueProvider struct {
	client sqs.Client
}

func NewQueueProvider(client sqs.Client) (QueueProvider, error) {
	return QueueProvider{
		client: client,
	}, nil
}
