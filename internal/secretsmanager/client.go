package secretsmanager

import (
	"context"
	"encoding/base64"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Secret struct {
	Name        string
	ARN         string
	Description string
	LastChanged string
}

type Client struct {
	client *secretsmanager.Client
}

func NewClient(cfg aws.Config) *Client {
	return &Client{client: secretsmanager.NewFromConfig(cfg)}
}

func (c *Client) ListSecrets(ctx context.Context) ([]Secret, error) {
	var secrets []Secret
	paginator := secretsmanager.NewListSecretsPaginator(c.client, &secretsmanager.ListSecretsInput{
		MaxResults: aws.Int32(100),
	})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, s := range out.SecretList {
			secret := Secret{
				Name:        aws.ToString(s.Name),
				ARN:         aws.ToString(s.ARN),
				Description: aws.ToString(s.Description),
			}
			if s.LastChangedDate != nil {
				secret.LastChanged = s.LastChangedDate.Local().Format("2006-01-02 15:04:05")
			}
			secrets = append(secrets, secret)
		}
	}
	sort.Slice(secrets, func(i, j int) bool { return secrets[i].Name < secrets[j].Name })
	return secrets, nil
}

// GetSecretValue returns the current (AWSCURRENT) value of the secret.
// Binary secrets are returned base64-encoded.
func (c *Client) GetSecretValue(ctx context.Context, name string) (string, error) {
	out, err := c.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	})
	if err != nil {
		return "", err
	}
	if out.SecretString != nil {
		return aws.ToString(out.SecretString), nil
	}
	return base64.StdEncoding.EncodeToString(out.SecretBinary), nil
}
