//go:build example
// +build example

package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

var region, endpoint string

func init() {
	flag.StringVar(&region, "region", "us-west-2", "Region where the test should be run")
	flag.StringVar(&endpoint, "endpoint", "", "SQS endpoint. SDK default will be used if it's not provided")
}

var queue_name = uuid.New().String()

var sess *session.Session

var queue_url string

func TestMain(m *testing.M) {
	flag.Parse()
	fmt.Printf("tests running with region: %q\t endpoint: %q\n", region, endpoint)
	os.Setenv("AWS_REGION", region)
	os.Setenv(SQS_ENDPOINT_VAR, endpoint)
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", os.Getenv("TOD_CUSTOMER_CREDENTIAL_PATH"))
	sess, _ = CreateSession()
	os.Exit(m.Run())
}

func TestProtocol(t *testing.T) {
	assert.Equal(t, GetClientInfo(sess).JSONVersion, "1.0")
}

func TestCreateQueue(t *testing.T) {
	result, err := CreateQueue(sess, &queue_name)
	assert.True(t, err == nil, "Error: %s", err)
	queue_url = *result.QueueUrl
	assert.True(t, strings.Contains(queue_url, endpoint), "QueueUrl %s doesn't contain expected endpoint %s", queue_url, endpoint)
	assert.True(t, strings.Contains(queue_url, queue_name), "QueueUrl doesn't contain expected queue name %s", queue_name)
}

func TestMessageOperations(t *testing.T) {
	result, err := SendReceiveAndDeleteMessage(sess, &queue_url)
	assert.True(t, err == nil, "Error: %s", err)
	assert.Equal(t, result.SentMessageMd5, OriginalMessageMd5)
	assert.Equal(t, result.ReceivedMessagBody, OriginalMessage)
}

func TestListQueues(t *testing.T) {
	anotherQueue := "another-new-queue"
	CreateQueue(sess, &anotherQueue)
	result, err := ListQueues(sess, &queue_name)
	assert.True(t, err == nil, "Error: %s", err)
	queueUrls := result.QueueUrls
	assert.Equal(t, len(queueUrls), 1)
	assert.True(t, strings.Contains(*(queueUrls[0]), queue_name), "QueueUrl %s doesn't contain %s", *(queueUrls[0]), queue_name)
}

func TestNonExistentQueue(t *testing.T) {
	invalid_queue := "invalid_queue_that_doesnt_exist"
	_, err := SendReceiveAndDeleteMessage(sess, &invalid_queue)
	assertError(t, err, "AWS.SimpleQueueService.NonExistentQueue", 400, "does not exist")
}

func TestSyntheticException(t *testing.T) {
	err := SendMessageWithInvalidInput(sess, &queue_url)
	assertError(t, err, "InvalidParameterValue", 400, "not valid for this queue type")
}

func TestUnmappedErrorCode(t *testing.T) {
	err := DeleteMessageWithInvalidInput(sess, &queue_url)
	assertError(t, err, "ReceiptHandleIsInvalid", 404, "receipt handle is invalid")
}

func TestDeleteQueue(t *testing.T) {
	_, err := DeleteQueue(sess, &queue_url)
	assert.True(t, err == nil, "Error: %s", err)
}

func assertError(t *testing.T, err error, errorCode string, statusCode int, message string) {
	assert.True(t, err != nil, "Error: %s", err)
	actual_error_code := err.(awserr.Error).Code()
	assert.Equal(t, errorCode, actual_error_code)
	actual_status_code := err.(awserr.RequestFailure).StatusCode()
	assert.Equal(t, statusCode, actual_status_code)
	error_message := err.(awserr.Error).Message()
	assert.True(t, strings.Contains(error_message, message), "Message: \"%s\" doesn't contain expected \"%s\"", error_message, message)
	assert.True(t, err.(awserr.RequestFailure).RequestID() != "", "Expecting a non-empty RequestID")
}
