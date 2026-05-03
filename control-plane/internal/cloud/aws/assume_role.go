package aws

import (
	"context"
	"fmt"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Credentials are short-lived AWS credentials suitable for shelling out to
// `terraform` via environment variables.
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

// AsEnv returns the credentials as terraform-compatible environment variables.
func (c Credentials) AsEnv() []string {
	return []string{
		"AWS_ACCESS_KEY_ID=" + c.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY=" + c.SecretAccessKey,
		"AWS_SESSION_TOKEN=" + c.SessionToken,
	}
}

// AssumeRole calls STS AssumeRole on the user's account and returns short-lived
// credentials. The control plane's own credentials are read from the default
// AWS chain (env, shared config, IRSA, ...).
//
// roleARN is the ARN of the role in the user's account that has been
// configured to trust the ilabhu control plane. externalID is the per-user
// secret that prevents the confused deputy attack.
func AssumeRole(ctx context.Context, roleARN, externalID, sessionName string, duration time.Duration) (*Credentials, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load default aws config: %w", err)
	}
	client := sts.NewFromConfig(cfg)

	input := &sts.AssumeRoleInput{
		RoleArn:         awsv2.String(roleARN),
		RoleSessionName: awsv2.String(sessionName),
		DurationSeconds: awsv2.Int32(int32(duration.Seconds())),
	}
	if externalID != "" {
		input.ExternalId = awsv2.String(externalID)
	}

	out, err := client.AssumeRole(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("sts:AssumeRole: %w", err)
	}
	c := out.Credentials
	return &Credentials{
		AccessKeyID:     awsv2.ToString(c.AccessKeyId),
		SecretAccessKey: awsv2.ToString(c.SecretAccessKey),
		SessionToken:    awsv2.ToString(c.SessionToken),
		Expiration:      awsv2.ToTime(c.Expiration),
	}, nil
}
