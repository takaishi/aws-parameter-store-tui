package secretsmanager

import (
	"context"

	"github.com/takaishi/aws-tui/internal/ui"
)

type backend struct {
	client *Client
}

func (b *backend) ServiceName() string { return "AWS Secrets Manager" }
func (b *backend) ItemNoun() string    { return "secrets" }

func (b *backend) List(ctx context.Context) ([]ui.Item, error) {
	secrets, err := b.client.ListSecrets(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]ui.Item, 0, len(secrets))
	for _, s := range secrets {
		items = append(items, ui.Item{
			Name:      s.Name,
			Meta:      s.LastChanged,
			Sensitive: true,
			Fields: []ui.Field{
				{Label: "Name", Value: s.Name},
				{Label: "ARN", Value: s.ARN},
				{Label: "Description", Value: s.Description},
				{Label: "Last Changed", Value: s.LastChanged},
			},
		})
	}
	return items, nil
}

func (b *backend) GetValue(ctx context.Context, name string) (string, error) {
	return b.client.GetSecretValue(ctx, name)
}
