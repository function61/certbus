package main

import (
	"bytes"
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/function61/gokit/aws/s3facade"
	"github.com/go-acme/lego/v4/challenge"
)

// presents a ACME challenge token on a webserver by writing it in an S3 bucket
type bucketChallengeUploader struct {
	challengesBucket *s3facade.BucketContext
}

var _ challenge.Provider = (*bucketChallengeUploader)(nil)

func (h *bucketChallengeUploader) Present(domain string, token string, keyAuth string) error {
	// once we've written the file and returned "ok" to our caller, ACME servers will send request to
	// http://DOMAIN_TO_VALIDATE/.well-known/acme-challenge/TOKEN
	_, err := h.challengesBucket.S3.PutObjectWithContext(context.Background(), &s3.PutObjectInput{
		Bucket: h.challengesBucket.Name,
		Key:    aws.String("acme-challenge/" + token),
		Body:   bytes.NewReader([]byte(keyAuth)),
	})
	return err
}

func (h *bucketChallengeUploader) CleanUp(domain string, token string, keyAuth string) error {
	// using S3 auto-delete, so theoretically no need to delete the file.
	// but let's still try to be good citizens.
	_, err := h.challengesBucket.S3.DeleteObjectWithContext(context.Background(), &s3.DeleteObjectInput{
		Bucket: h.challengesBucket.Name,
		Key:    aws.String("acme-challenge/" + token),
	})
	return err
}
