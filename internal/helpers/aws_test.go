package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	. "github.com/onsi/gomega"
)

const (
	fakeArn             = "arn:aws:sts::123456789012:assumed-role/xaccounts3access/s3-access-example"
	fakeSession         = "s3-access-example"
	fakeAssumedRoleId   = "AROA3XFRBF535PLBIFPI4:s3-access-example"
	fakeSecretAccessKey = "9drTJvcXLB89EXAMPLELB8923FB892xMFI"
	fakeSessionToken    = "AQoXdzELDDY//////////wEaoAK1wvxJY12r2IrDFT2IvAzTCn3zHoZ7YNtpiQLF0MqZye/qwjzP2iEXAMPLEbw/m3hsj8VBTkPORGvr9jM5sgP+w9IZWZnU+LWhmg+a5fDi2oTGUYcdg9uexQ4mtCHIHfi4citgqZTgco40Yqr4lIlo4V2b2Dyauk0eYFNebHtYlFVgAUj+7Indz3LU0aTWk1WKIjHmmMCIoTkyYp/k7kUG7moeEYKSitwQIi6Gjn+nyzM+PtoA3685ixzv0R7i5rjQi0YE0lf1oeie3bDiNHncmzosRM6SFiPzSvp6h/32xQuZsjcypmwsPSDtTPYcs0+YN/8BRi2/IcrxSpnWEXAMPLEXSDFTAQAM6Dl9zR0tXoybnlrZIwMLlMi1Kcgo5OytwU="
	fakeAccessKeyId     = "ASIAJEXAMPLEXEG2JICEA"
)

type FakeSTSClient struct{}

func timeNow() time.Time {
	return time.Date(2020, 11, 12, 0, 0, 0, 0, time.UTC)
}

func (f FakeSTSClient) AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {

	roleArn := aws.ToString(params.RoleArn)
	if !arn.IsARN(roleArn) {
		return nil, fmt.Errorf("Invalid ARN: %v", roleArn)
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

func TestAssumeRole(t *testing.T) {
	testCases := []struct {
		name        string
		role        string
		session     string
		expectedArn string
		expectedErr bool
	}{
		{
			name:        "missing role",
			role:        "",
			session:     fakeSession,
			expectedArn: "",
			expectedErr: true,
		},
		{
			name:        "invalid role",
			role:        "my_invalid_role",
			session:     fakeSession,
			expectedArn: "",
			expectedErr: true,
		},
		{
			name:        "missing session",
			role:        fakeArn,
			session:     "",
			expectedArn: "",
			expectedErr: true,
		},
	}

	for _, tt := range testCases {
		ctx := context.Background()
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			gotArn, gotErr := AssumeRole(ctx, tt.role, tt.session)
			g.Expect(gotArn).To(Equal(tt.expectedArn))
			if (gotErr != nil) != tt.expectedErr {
				t.Errorf("AssumeRole error = %v, expectedErr = %v", gotErr, tt.expectedErr)
			}
		})
	}
}

func TestTakeRole(t *testing.T) {
	testCases := []struct {
		name           string
		input          *sts.AssumeRoleInput
		expectedOutput *sts.AssumeRoleOutput
		expectedErr    bool
	}{
		{
			name: "test valid ARN",
			input: &sts.AssumeRoleInput{
				RoleArn:         aws.String(fakeArn),
				RoleSessionName: aws.String(fakeSession),
			},
			expectedOutput: &sts.AssumeRoleOutput{
				AssumedRoleUser: &types.AssumedRoleUser{
					Arn:           aws.String(fakeArn),
					AssumedRoleId: aws.String(fakeAssumedRoleId),
				},
				Credentials: &types.Credentials{
					SecretAccessKey: aws.String(fakeSecretAccessKey),
					SessionToken:    aws.String(fakeSessionToken),
					Expiration:      aws.Time(timeNow()),
					AccessKeyId:     aws.String(fakeAccessKeyId),
				},
			},
			expectedErr: false,
		},
		{
			name: "test invalid ARN",
			input: &sts.AssumeRoleInput{
				RoleArn:         aws.String("invalid-arn"),
				RoleSessionName: aws.String(fakeSession),
			},
			expectedOutput: nil,
			expectedErr:    true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			gotOutput, gotErr := TakeRole(context.TODO(), FakeSTSClient{}, tt.input)
			g.Expect(gotOutput).To(Equal(tt.expectedOutput))
			if (gotErr != nil) != tt.expectedErr {
				t.Errorf("TakeRole error = %v, expectedErr = %v", gotErr, tt.expectedErr)
			}
		})
	}
}
