package cloudwatchlogs

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type Client struct {
	client *awscwl.Client
}

func NewClient(cfg aws.Config) *Client {
	return &Client{client: awscwl.NewFromConfig(cfg)}
}

func (c *Client) ListLogGroups(ctx context.Context) ([]types.LogGroup, error) {
	var groups []types.LogGroup
	paginator := awscwl.NewDescribeLogGroupsPaginator(c.client, &awscwl.DescribeLogGroupsInput{})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		groups = append(groups, out.LogGroups...)
	}
	sort.Slice(groups, func(i, j int) bool {
		return aws.ToString(groups[i].LogGroupName) < aws.ToString(groups[j].LogGroupName)
	})
	return groups, nil
}

// ListLogStreams returns the log group's streams, most recently active
// first.
func (c *Client) ListLogStreams(ctx context.Context, logGroupName string) ([]types.LogStream, error) {
	var streams []types.LogStream
	paginator := awscwl.NewDescribeLogStreamsPaginator(c.client, &awscwl.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroupName),
		OrderBy:      types.OrderByLastEventTime,
		Descending:   aws.Bool(true),
	})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		streams = append(streams, out.LogStreams...)
	}
	return streams, nil
}

// GetRecentEvents returns the most recent log events in the stream, oldest
// first.
func (c *Client) GetRecentEvents(ctx context.Context, logGroupName, logStreamName string) ([]types.OutputLogEvent, error) {
	out, err := c.client.GetLogEvents(ctx, &awscwl.GetLogEventsInput{
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String(logStreamName),
		Limit:         aws.Int32(200),
		StartFromHead: aws.Bool(false),
	})
	if err != nil {
		return nil, err
	}
	return out.Events, nil
}
