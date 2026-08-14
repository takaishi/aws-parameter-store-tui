package securitygroup

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/takaishi/aws-tui/internal/ui"
)

func newScreen(c *Client, region string) *ui.Screen {
	return &ui.Screen{
		Title: fmt.Sprintf("Amazon EC2 Security Groups (%s)", region),
		Noun:  "security groups",
		List: func(ctx context.Context) ([]ui.Item, error) {
			groups, err := c.ListSecurityGroups(ctx)
			if err != nil {
				return nil, err
			}
			nameByID := make(map[string]string, len(groups))
			for _, g := range groups {
				nameByID[aws.ToString(g.GroupId)] = aws.ToString(g.GroupName)
			}
			items := make([]ui.Item, 0, len(groups))
			for _, g := range groups {
				items = append(items, securityGroupItem(g, nameByID))
			}
			return items, nil
		},
	}
}

func securityGroupItem(g types.SecurityGroup, nameByID map[string]string) ui.Item {
	name := aws.ToString(g.GroupName)
	meta := aws.ToString(g.GroupId)
	if vpc := aws.ToString(g.VpcId); vpc != "" {
		meta += ", " + vpc
	}
	return ui.Item{
		Name:       name,
		Meta:       meta,
		CopyValue:  aws.ToString(g.GroupId),
		Fields:     securityGroupFields(g),
		ValueLabel: "Rules",
		Value: func(ctx context.Context) (string, error) {
			return formatRules(g, nameByID), nil
		},
	}
}

func securityGroupFields(g types.SecurityGroup) []ui.Field {
	fields := []ui.Field{
		{Label: "Name", Value: aws.ToString(g.GroupName)},
		{Label: "Group ID", Value: aws.ToString(g.GroupId)},
		{Label: "VPC ID", Value: aws.ToString(g.VpcId)},
		{Label: "Owner ID", Value: aws.ToString(g.OwnerId)},
		{Label: "Description", Value: aws.ToString(g.Description)},
	}
	if len(g.Tags) > 0 {
		fields = append(fields, ui.Field{Label: "Tags", Value: formatTags(g.Tags)})
	}
	return fields
}

func formatTags(tags []types.Tag) string {
	pairs := make([]string, 0, len(tags))
	for _, t := range tags {
		pairs = append(pairs, fmt.Sprintf("%s=%s", aws.ToString(t.Key), aws.ToString(t.Value)))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ", ")
}

func formatRules(g types.SecurityGroup, nameByID map[string]string) string {
	var b strings.Builder
	b.WriteString("Inbound:\n")
	if len(g.IpPermissions) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, p := range g.IpPermissions {
		writePermission(&b, p, nameByID)
	}
	b.WriteString("\nOutbound:\n")
	if len(g.IpPermissionsEgress) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, p := range g.IpPermissionsEgress {
		writePermission(&b, p, nameByID)
	}
	return strings.TrimRight(b.String(), "\n")
}

func writePermission(b *strings.Builder, p types.IpPermission, nameByID map[string]string) {
	proto := aws.ToString(p.IpProtocol)
	portRange := formatPortRange(proto, p.FromPort, p.ToPort)
	sources := permissionSources(p, nameByID)
	if len(sources) == 0 {
		fmt.Fprintf(b, "  %s %s\n", proto, portRange)
		return
	}
	for _, src := range sources {
		fmt.Fprintf(b, "  %s %s  %s\n", proto, portRange, src)
	}
}

func permissionSources(p types.IpPermission, nameByID map[string]string) []string {
	var sources []string
	for _, r := range p.IpRanges {
		s := aws.ToString(r.CidrIp)
		if desc := aws.ToString(r.Description); desc != "" {
			s += " (" + desc + ")"
		}
		sources = append(sources, s)
	}
	for _, r := range p.Ipv6Ranges {
		s := aws.ToString(r.CidrIpv6)
		if desc := aws.ToString(r.Description); desc != "" {
			s += " (" + desc + ")"
		}
		sources = append(sources, s)
	}
	for _, pl := range p.PrefixListIds {
		s := aws.ToString(pl.PrefixListId)
		if desc := aws.ToString(pl.Description); desc != "" {
			s += " (" + desc + ")"
		}
		sources = append(sources, s)
	}
	for _, ug := range p.UserIdGroupPairs {
		id := aws.ToString(ug.GroupId)
		s := id
		name := aws.ToString(ug.GroupName)
		if name == "" {
			name = nameByID[id]
		}
		if name != "" {
			s += " (" + name + ")"
		}
		sources = append(sources, s)
	}
	return sources
}

func formatPortRange(proto string, from, to *int32) string {
	if proto == "-1" {
		return "all traffic"
	}
	f, t := aws.ToInt32(from), aws.ToInt32(to)
	if f == -1 && t == -1 {
		return "all"
	}
	if f == t {
		return fmt.Sprintf("%d", f)
	}
	return fmt.Sprintf("%d-%d", f, t)
}
