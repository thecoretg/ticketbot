package amz

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func GetAndUnmarshalParam(ctx context.Context, s *ssm.Client, paramName string, withDecryption bool, v any) error {
	p, err := GetParam(ctx, s, paramName, withDecryption)
	if err != nil {
		return fmt.Errorf("getting param: %w", err)
	}

	return UnmarshalParam(p, v)
}

// GetParam returns the output of an SSM Get Parameter call
func GetParam(ctx context.Context, s *ssm.Client, paramName string, withDecryption bool) (*ssm.GetParameterOutput, error) {
	input := &ssm.GetParameterInput{
		Name:           aws.String(paramName),
		WithDecryption: aws.Bool(withDecryption),
	}

	return s.GetParameter(ctx, input)
}

// UnmarshalParam unmarshals an SSM Get Parameter output into a struct
func UnmarshalParam(output *ssm.GetParameterOutput, v any) error {
	if err := json.Unmarshal([]byte(*output.Parameter.Value), v); err != nil {
		return fmt.Errorf("unmarshaling to struct: %w", err)
	}

	return nil
}
