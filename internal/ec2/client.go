package ec2

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Client struct {
	client *awsec2.Client
}

func NewClient(cfg aws.Config) *Client {
	return &Client{client: awsec2.NewFromConfig(cfg)}
}

func (c *Client) ListInstances(ctx context.Context) ([]types.Instance, error) {
	var instances []types.Instance
	paginator := awsec2.NewDescribeInstancesPaginator(c.client, &awsec2.DescribeInstancesInput{})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range out.Reservations {
			instances = append(instances, r.Instances...)
		}
	}
	sort.Slice(instances, func(i, j int) bool {
		return instanceName(instances[i]) < instanceName(instances[j])
	})
	return instances, nil
}

func instanceName(i types.Instance) string {
	for _, t := range i.Tags {
		if aws.ToString(t.Key) == "Name" {
			return aws.ToString(t.Value)
		}
	}
	return aws.ToString(i.InstanceId)
}
