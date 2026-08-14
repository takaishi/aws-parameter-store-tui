package secretsmanager

import (
	"context"
	"fmt"

	"github.com/takaishi/aws-tui/internal/ui"
)

func newScreen(client *Client, region string) *ui.Screen {
	return &ui.Screen{
		Title: fmt.Sprintf("AWS Secrets Manager (%s)", region),
		Noun:  "secrets",
		List: func(ctx context.Context) ([]ui.Item, error) {
			secrets, err := client.ListSecrets(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]ui.Item, 0, len(secrets))
			for _, s := range secrets {
				name := s.Name
				items = append(items, ui.Item{
					Name:      name,
					Meta:      s.LastChanged,
					Sensitive: true,
					Fields: []ui.Field{
						{Label: "Name", Value: s.Name},
						{Label: "ARN", Value: s.ARN},
						{Label: "Description", Value: s.Description},
						{Label: "Last Changed", Value: s.LastChanged},
					},
					Value: func(ctx context.Context) (string, error) {
						return client.GetSecretValue(ctx, name)
					},
				})
			}
			return items, nil
		},
	}
}
