package securitygroup

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Client struct {
	client *ec2.Client
}

func NewClient(cfg aws.Config) *Client {
	return &Client{client: ec2.NewFromConfig(cfg)}
}

func (c *Client) ListSecurityGroups(ctx context.Context) ([]types.SecurityGroup, error) {
	var groups []types.SecurityGroup
	paginator := ec2.NewDescribeSecurityGroupsPaginator(c.client, &ec2.DescribeSecurityGroupsInput{})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		groups = append(groups, out.SecurityGroups...)
	}
	sort.Slice(groups, func(i, j int) bool {
		return aws.ToString(groups[i].GroupName) < aws.ToString(groups[j].GroupName)
	})
	return groups, nil
}
