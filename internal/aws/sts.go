package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Code modified from https://aws.github.io/aws-sdk-go-v2/docs/code-examples/sts/assumerole/

type STSApi interface {
	AssumeRole(ctx context.Context,
		params *sts.AssumeRoleInput,
		optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

func STSAssumeRole(ctx context.Context, client STSApi, role string, session string) (string, error) {

	if role == "" {
		return "", fmt.Errorf("role %v is empty", role)
	}
	if !arn.IsARN(role) {
		return "", fmt.Errorf("role %v is not a valid ARN", role)
	}
	if session == "" {
		return "", fmt.Errorf("session %v is empty", session)
	}

	input := &sts.AssumeRoleInput{
		RoleArn:         &role,
		RoleSessionName: &session,
	}

	result, err := client.AssumeRole(ctx, input)
	if err != nil {
		return "", err
	}

	return *result.AssumedRoleUser.AssumedRoleId, nil
}

func STSGetCallerIdentity(ctx context.Context, client STSApi) (sts.GetCallerIdentityOutput, error) {
	input := &sts.GetCallerIdentityInput{}
	result, err := client.GetCallerIdentity(ctx, input)
	if err != nil {
		return sts.GetCallerIdentityOutput{}, err
	}
	return *result, nil
}
