package ec2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/takaishi/aws-tui/internal/ui"
)

func newScreen(c *Client, region string) *ui.Screen {
	return &ui.Screen{
		Title: fmt.Sprintf("Amazon EC2 (%s)", region),
		Noun:  "instances",
		List: func(ctx context.Context) ([]ui.Item, error) {
			instances, err := c.ListInstances(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]ui.Item, 0, len(instances))
			for _, i := range instances {
				items = append(items, instanceItem(i))
			}
			return items, nil
		},
	}
}

func instanceItem(i types.Instance) ui.Item {
	name := instanceName(i)
	state := ""
	if i.State != nil {
		state = string(i.State.Name)
	}
	meta := fmt.Sprintf("%s, %s, %s", aws.ToString(i.InstanceId), string(i.InstanceType), state)
	return ui.Item{
		Name:       name,
		Meta:       meta,
		CopyValue:  aws.ToString(i.InstanceId),
		Fields:     instanceFields(i),
		ValueLabel: "Tags",
		Value: func(ctx context.Context) (string, error) {
			return formatTags(i.Tags), nil
		},
	}
}

func instanceFields(i types.Instance) []ui.Field {
	state := ""
	if i.State != nil {
		state = string(i.State.Name)
	}
	fields := []ui.Field{
		{Label: "Instance ID", Value: aws.ToString(i.InstanceId)},
		{Label: "Instance Type", Value: string(i.InstanceType)},
		{Label: "State", Value: state},
		{Label: "AMI", Value: aws.ToString(i.ImageId)},
	}
	if i.Placement != nil {
		fields = append(fields, ui.Field{Label: "Availability Zone", Value: aws.ToString(i.Placement.AvailabilityZone)})
	}
	if ip := aws.ToString(i.PrivateIpAddress); ip != "" {
		fields = append(fields, ui.Field{Label: "Private IP", Value: ip})
	}
	if ip := aws.ToString(i.PublicIpAddress); ip != "" {
		fields = append(fields, ui.Field{Label: "Public IP", Value: ip})
	}
	if dns := aws.ToString(i.PrivateDnsName); dns != "" {
		fields = append(fields, ui.Field{Label: "Private DNS", Value: dns})
	}
	if dns := aws.ToString(i.PublicDnsName); dns != "" {
		fields = append(fields, ui.Field{Label: "Public DNS", Value: dns})
	}
	if vpc := aws.ToString(i.VpcId); vpc != "" {
		fields = append(fields, ui.Field{Label: "VPC ID", Value: vpc})
	}
	if subnet := aws.ToString(i.SubnetId); subnet != "" {
		fields = append(fields, ui.Field{Label: "Subnet ID", Value: subnet})
	}
	if key := aws.ToString(i.KeyName); key != "" {
		fields = append(fields, ui.Field{Label: "Key Pair", Value: key})
	}
	if len(i.SecurityGroups) > 0 {
		fields = append(fields, ui.Field{Label: "Security Groups", Value: formatSecurityGroups(i.SecurityGroups)})
	}
	if i.LaunchTime != nil {
		fields = append(fields, ui.Field{Label: "Launch Time", Value: fmtTime(i.LaunchTime)})
	}
	if reason := aws.ToString(i.StateTransitionReason); reason != "" {
		fields = append(fields, ui.Field{Label: "State Transition Reason", Value: reason})
	}
	return fields
}

func formatSecurityGroups(groups []types.GroupIdentifier) string {
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		parts = append(parts, fmt.Sprintf("%s (%s)", aws.ToString(g.GroupName), aws.ToString(g.GroupId)))
	}
	return strings.Join(parts, ", ")
}

// formatTags renders tags as a JSON object so the detail view shows them as
// a key/value list (encoding/json sorts map keys alphabetically).
func formatTags(tags []types.Tag) string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
