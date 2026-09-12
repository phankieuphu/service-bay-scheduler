package consumer

import (
	"context"
	internal_config "customer-service/config"
	"customer-service/internal/domain/ports"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type CustomerConsumer struct {
	cfg                  *internal_config.Config
	entryCustomerService ports.CustomerService
	queueURL             string
	provider             QueueProvider
}

func NewCustomerConsumer(ctx context.Context, provider QueueProvider, cfg *internal_config.Config, entryCustomerService ports.CustomerService, queueURL string) (ports.IConsumer, error) {
	return &CustomerConsumer{
		cfg:                  cfg,
		entryCustomerService: entryCustomerService,
		queueURL:             queueURL,
		provider:             provider,
	}, nil

}

func (a *CustomerConsumer) Start(ctx context.Context) {

	queueURL := a.queueURL

	log.Println("SQS consumer started...")

	for {
		resp, err := a.provider.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &queueURL,
			MaxNumberOfMessages: 5,
			WaitTimeSeconds:     20, // long polling
			VisibilityTimeout:   30,
		})
		if err != nil {
			log.Println("receive message error:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if len(resp.Messages) == 0 {
			continue
		}

		for _, msg := range resp.Messages {
			err := a.ProcessMessage(ctx, msg)
			if err != nil {
				log.Println("processing failed:", err)
				continue
			}
		}
	}
}

func (a *CustomerConsumer) ProcessMessage(
	ctx context.Context,
	msg types.Message,
) error {
	log.Println("📩 raw SQS message received")

	// 1. Unwrap SNS envelope

	//if err := json.Unmarshal([]byte(*msg.Body), &snsMsg); err != nil {
	//	return err
	//}
	//
	//a.entryCustomerService.Save(ctx, snsMsg.Message)

	// 5. Delete message after success
	return a.DeleteMessage(ctx, *msg.ReceiptHandle)
}

func (a *CustomerConsumer) DeleteMessage(
	ctx context.Context,
	receiptHandle string,
) error {
	queueURL := a.queueURL
	_, err := a.provider.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &queueURL,
		ReceiptHandle: &receiptHandle,
	})

	if err == nil {
		log.Println("message deleted")
	}

	return err
}
