package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

const (
	fakeArn             = "arn:aws:iam::123456789012:role/my_role"
	fakeSession         = "s3-access-example"
	fakeAssumedRoleId   = "AROA3XFRBF535PLBIFPI4:s3-access-example"
	fakeSecretAccessKey = "9drTJvcXLB89EXAMPLELB8923FB892xMFI"
	fakeSessionToken    = "AQoXdzELDDY//////////wEaoAK1wvxJY12r2IrDFT2IvAzTCn3zHoZ7YNtpiQLF0MqZye/qwjzP2iEXAMPLEbw/m3hsj8VBTkPORGvr9jM5sgP+w9IZWZnU+LWhmg+a5fDi2oTGUYcdg9uexQ4mtCHIHfi4citgqZTgco40Yqr4lIlo4V2b2Dyauk0eYFNebHtYlFVgAUj+7Indz3LU0aTWk1WKIjHmmMCIoTkyYp/k7kUG7moeEYKSitwQIi6Gjn+nyzM+PtoA3685ixzv0R7i5rjQi0YE0lf1oeie3bDiNHncmzosRM6SFiPzSvp6h/32xQuZsjcypmwsPSDtTPYcs0+YN/8BRi2/IcrxSpnWEXAMPLEXSDFTAQAM6Dl9zR0tXoybnlrZIwMLlMi1Kcgo5OytwU="
	fakeAccessKeyId     = "ASIAJEXAMPLEXEG2JICEA"
)

func timeNow() time.Time {
	return time.Date(2020, 11, 12, 0, 0, 0, 0, time.UTC)
}

type FakeSTSClient struct{}

func (f FakeSTSClient) AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {

	roleArn := aws.ToString(params.RoleArn)
	if !arn.IsARN(roleArn) {
		return nil, fmt.Errorf("invalid ARN: %v", roleArn)
	}

	return &sts.AssumeRoleOutput{
		AssumedRoleUser: &types.AssumedRoleUser{
			Arn:           params.RoleArn,
			AssumedRoleId: aws.String(fakeAssumedRoleId),
		},
		Credentials: &types.Credentials{
			SecretAccessKey: aws.String(fakeSecretAccessKey),
			SessionToken:    aws.String(fakeSessionToken),
			Expiration:      aws.Time(timeNow()),
			AccessKeyId:     aws.String(fakeAccessKeyId),
		},
	}, nil
}

func GetSTSClient() (*sts.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	client := sts.NewFromConfig(cfg)
	return client, nil
}
