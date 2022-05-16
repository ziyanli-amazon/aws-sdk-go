//go:build example
// +build example

package main

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"os"
)

const SQS_ENDPOINT_VAR = "SQS_ENDPOINT"

var OriginalMessage = "a-message"

var OriginalMessageMd5 = md5String(OriginalMessage)

func CreateSession() (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Endpoint: aws.String(os.Getenv(SQS_ENDPOINT_VAR)),
	})
}

func GetClientInfo(sess *session.Session) metadata.ClientInfo {
	svc := *sqs.New(sess)
	return svc.Client.ClientInfo
}

func CreateQueue(sess *session.Session, queue *string) (*sqs.CreateQueueOutput, error) {
	svc := *sqs.New(sess)
	result, err := svc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: queue,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ListQueues(sess *session.Session, prefix *string) (*sqs.ListQueuesOutput, error) {
	svc := *sqs.New(sess)
	result, err := svc.ListQueues(&sqs.ListQueuesInput{
		QueueNamePrefix: prefix,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func DeleteQueue(sess *session.Session, queue_url *string) (*sqs.DeleteQueueOutput, error) {
	svc := *sqs.New(sess)
	result, err := svc.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: queue_url,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type SendAndReceiveResult struct {
	SentMessageMd5     string
	ReceivedMessagBody string
}

func SendReceiveAndDeleteMessage(sess *session.Session, queue_url *string) (*SendAndReceiveResult, error) {
	svc := *sqs.New(sess)
	result, err := svc.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    queue_url,
		MessageBody: &OriginalMessage,
	})
	if err == nil {
		body, err := receiveAndDeleteMessage(svc, queue_url)
		return &SendAndReceiveResult{SentMessageMd5: *result.MD5OfMessageBody, ReceivedMessagBody: *body}, err
	}
	return nil, err
}

func SendMessageWithInvalidInput(sess *session.Session, queue_url *string) error {
	svc := *sqs.New(sess)
	someId := "some-id"
	_, err := svc.SendMessage(&sqs.SendMessageInput{
		QueueUrl:               queue_url,
		MessageBody:            &OriginalMessage,
		MessageDeduplicationId: &someId,
	})
	return err
}

func DeleteMessageWithInvalidInput(sess *session.Session, queue_url *string) error {
	svc := *sqs.New(sess)
	someHandle := "some-handle"
	return deleteMessage(svc, queue_url, &someHandle)
}

func receiveAndDeleteMessage(svc sqs.SQS, queue_url *string) (*string, error) {
	max_wait := int64(3)
	result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:        queue_url,
		WaitTimeSeconds: &max_wait,
	})
	if err == nil {
		receipt_handle := result.Messages[0].ReceiptHandle
		return result.Messages[0].Body, deleteMessage(svc, queue_url, receipt_handle)
	}
	return nil, err
}

func deleteMessage(svc sqs.SQS, queue_url *string, receipt_handle *string) error {
	_, err := svc.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      queue_url,
		ReceiptHandle: receipt_handle,
	})
	return err
}

func md5String(message string) string {
	data := []byte(message)
	md5_sum := md5.Sum(data)
	return hex.EncodeToString(md5_sum[:])
}
