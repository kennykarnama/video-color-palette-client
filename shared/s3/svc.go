package s3

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	s3cli "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/aws/awserr"
	pkgError "github.com/pkg/errors"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	svc        *s3cli.S3
)

const (
	ErrCodeNotFound = "NotFound"
)

func init() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-southeast-1"),
	}))
	uploader = s3manager.NewUploader(sess)
	downloader = s3manager.NewDownloader(sess)
	svc = s3cli.New(sess)
}

func CheckKeyExist(ctx context.Context, bucket string, key string) (bool, error) {
	_, err := HeadObject(bucket, key)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ErrCodeNotFound:
				return false, nil
			default:
				return false, pkgError.Wrap(err, "CheckKeyExist failed")
			}
		}
		return false, pkgError.Wrap(err, "CheckKeyExist failed")
	}
	return true, nil
}

func HeadObject(bucket, key string) (*s3.HeadObjectOutput, error) {
	output, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return output, nil
}
