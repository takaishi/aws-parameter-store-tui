package cloudwatchlogs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/takaishi/aws-tui/internal/ui"
)

func logGroupsScreen(c *Client, region string) *ui.Screen {
	return &ui.Screen{
		Title: fmt.Sprintf("Amazon CloudWatch Logs (%s)", region),
		Noun:  "log groups",
		List: func(ctx context.Context) ([]ui.Item, error) {
			groups, err := c.ListLogGroups(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]ui.Item, 0, len(groups))
			for _, g := range groups {
				name := aws.ToString(g.LogGroupName)
				meta := "no expiry"
				if g.RetentionInDays != nil {
					meta = fmt.Sprintf("%d day retention", *g.RetentionInDays)
				}
				meta += fmt.Sprintf(", %s", formatBytes(aws.ToInt64(g.StoredBytes)))
				items = append(items, ui.Item{
					Name:      name,
					Meta:      meta,
					CopyValue: aws.ToString(g.LogGroupArn),
					Child: func() *ui.Screen {
						return logStreamsScreen(c, name, name)
					},
				})
			}
			return items, nil
		},
	}
}

func logStreamsScreen(c *Client, title, logGroupName string) *ui.Screen {
	return &ui.Screen{
		Title: title,
		Noun:  "log streams",
		List: func(ctx context.Context) ([]ui.Item, error) {
			streams, err := c.ListLogStreams(ctx, logGroupName)
			if err != nil {
				return nil, err
			}
			items := make([]ui.Item, 0, len(streams))
			for _, s := range streams {
				items = append(items, logStreamItem(c, logGroupName, s))
			}
			return items, nil
		},
	}
}

func logStreamItem(c *Client, logGroupName string, s types.LogStream) ui.Item {
	name := aws.ToString(s.LogStreamName)
	meta := "no events"
	if s.LastEventTimestamp != nil {
		meta = "last event " + fmtMillis(*s.LastEventTimestamp)
	}
	return ui.Item{
		Name:       name,
		Meta:       meta,
		CopyValue:  aws.ToString(s.Arn),
		Fields:     logStreamFields(s),
		ValueLabel: "Recent Events",
		Value: func(ctx context.Context) (string, error) {
			events, err := c.GetRecentEvents(ctx, logGroupName, name)
			if err != nil {
				return "", err
			}
			return formatEvents(events), nil
		},
	}
}

func logStreamFields(s types.LogStream) []ui.Field {
	fields := []ui.Field{
		{Label: "Log Stream", Value: aws.ToString(s.LogStreamName)},
	}
	if s.CreationTime != nil {
		fields = append(fields, ui.Field{Label: "Created", Value: fmtMillis(*s.CreationTime)})
	}
	if s.FirstEventTimestamp != nil {
		fields = append(fields, ui.Field{Label: "First Event", Value: fmtMillis(*s.FirstEventTimestamp)})
	}
	if s.LastEventTimestamp != nil {
		fields = append(fields, ui.Field{Label: "Last Event", Value: fmtMillis(*s.LastEventTimestamp)})
	}
	if s.LastIngestionTime != nil {
		fields = append(fields, ui.Field{Label: "Last Ingestion", Value: fmtMillis(*s.LastIngestionTime)})
	}
	fields = append(fields, ui.Field{Label: "Stored Bytes", Value: formatBytes(aws.ToInt64(s.StoredBytes))})
	return fields
}

func formatEvents(events []types.OutputLogEvent) string {
	if len(events) == 0 {
		return "(no events)"
	}
	var b strings.Builder
	for _, e := range events {
		fmt.Fprintf(&b, "%s  %s\n", fmtMillis(aws.ToInt64(e.Timestamp)), strings.TrimRight(aws.ToString(e.Message), "\n"))
	}
	return strings.TrimRight(b.String(), "\n")
}

func fmtMillis(ms int64) string {
	return time.UnixMilli(ms).Local().Format("2006-01-02 15:04:05")
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n2 := n / unit; n2 >= unit; n2 /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
