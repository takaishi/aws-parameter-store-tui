package route53

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsroute53 "github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type Client struct {
	client *awsroute53.Client
}

func NewClient(cfg aws.Config) *Client {
	return &Client{client: awsroute53.NewFromConfig(cfg)}
}

func (c *Client) ListHostedZones(ctx context.Context) ([]types.HostedZone, error) {
	var zones []types.HostedZone
	var marker *string
	for {
		out, err := c.client.ListHostedZones(ctx, &awsroute53.ListHostedZonesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}
		zones = append(zones, out.HostedZones...)
		if !out.IsTruncated {
			break
		}
		marker = out.NextMarker
	}
	sort.Slice(zones, func(i, j int) bool {
		return aws.ToString(zones[i].Name) < aws.ToString(zones[j].Name)
	})
	return zones, nil
}

func (c *Client) ListResourceRecordSets(ctx context.Context, hostedZoneID string) ([]types.ResourceRecordSet, error) {
	var records []types.ResourceRecordSet
	paginator := awsroute53.NewListResourceRecordSetsPaginator(c.client, &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
	})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		records = append(records, out.ResourceRecordSets...)
	}
	return records, nil
}
