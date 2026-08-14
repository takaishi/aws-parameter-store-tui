package parameterstore

import (
	"context"
	"strconv"

	"github.com/takaishi/aws-tui/internal/ui"
)

type backend struct {
	client *SSMClient
}

func (b *backend) ServiceName() string { return "AWS Parameter Store" }
func (b *backend) ItemNoun() string    { return "parameters" }

func (b *backend) List(ctx context.Context) ([]ui.Item, error) {
	params, err := b.client.ListParameters(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]ui.Item, 0, len(params))
	for _, p := range params {
		items = append(items, ui.Item{
			Name:      p.Name,
			Meta:      p.Type,
			Sensitive: p.Type == "SecureString",
			Fields: []ui.Field{
				{Label: "Name", Value: p.Name},
				{Label: "Type", Value: p.Type},
				{Label: "Version", Value: strconv.FormatInt(p.Version, 10)},
				{Label: "Last Modified", Value: p.LastModified},
			},
		})
	}
	return items, nil
}

func (b *backend) GetValue(ctx context.Context, name string) (string, error) {
	return b.client.GetParameterValue(ctx, name)
}
