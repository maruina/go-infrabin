package helpers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Code copied from https://aws.github.io/aws-sdk-go-v2/docs/code-examples/sts/assumerole/

// STSAssumeRoleAPI defines the interface for the AssumeRole function.
// We use this interface to test the function using a mocked service.
type STSApi interface {
	AssumeRole(ctx context.Context,
		params *sts.AssumeRoleInput,
		optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

// TakeRole gets temporary security credentials to access resources.
// Inputs:
//     c is the context of the method call, which includes the AWS Region.
//     api is the interface that defines the method call.
//     input defines the input arguments to the service call.
// Output:
//     If successful, an AssumeRoleOutput object containing the result of the service call and nil.
//     Otherwise, nil and an error from the call to AssumeRole.
func TakeRole(c context.Context, api STSApi, input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
	return api.AssumeRole(c, input)
}

func AssumeRole(ctx context.Context, role string, session string) (string, error) {

	if role == "" {
		return "", fmt.Errorf("")
	}
	if !arn.IsARN(role) {
		return "", fmt.Errorf("")
	}
	if session == "" {
		return "", fmt.Errorf("")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}

	client := sts.NewFromConfig(cfg)

	input := &sts.AssumeRoleInput{
		RoleArn:         &role,
		RoleSessionName: &session,
	}

	result, err := TakeRole(ctx, client, input)
	if err != nil {
		return "", err
	}

	return *result.AssumedRoleUser.Arn, nil
}
