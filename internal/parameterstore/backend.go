package parameterstore

import (
	"context"
	"fmt"
	"strconv"

	"github.com/takaishi/aws-tui/internal/ui"
)

func newScreen(client *SSMClient, region string) *ui.Screen {
	return &ui.Screen{
		Title: fmt.Sprintf("AWS Parameter Store (%s)", region),
		Noun:  "parameters",
		List: func(ctx context.Context) ([]ui.Item, error) {
			params, err := client.ListParameters(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]ui.Item, 0, len(params))
			for _, p := range params {
				name := p.Name
				items = append(items, ui.Item{
					Name:      name,
					Meta:      p.Type,
					Sensitive: p.Type == "SecureString",
					Fields: []ui.Field{
						{Label: "Name", Value: p.Name},
						{Label: "Type", Value: p.Type},
						{Label: "Version", Value: strconv.FormatInt(p.Version, 10)},
						{Label: "Last Modified", Value: p.LastModified},
					},
					Value: func(ctx context.Context) (string, error) {
						return client.GetParameterValue(ctx, name)
					},
				})
			}
			return items, nil
		},
	}
}
