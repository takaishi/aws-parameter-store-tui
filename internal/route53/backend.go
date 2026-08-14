package route53

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/takaishi/aws-tui/internal/ui"
)

func hostedZonesScreen(c *Client) *ui.Screen {
	return &ui.Screen{
		Title: "Amazon Route 53",
		Noun:  "hosted zones",
		List: func(ctx context.Context) ([]ui.Item, error) {
			zones, err := c.ListHostedZones(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]ui.Item, 0, len(zones))
			for _, z := range zones {
				id := shortZoneID(aws.ToString(z.Id))
				name := aws.ToString(z.Name)
				kind := "Public"
				if z.Config != nil && z.Config.PrivateZone {
					kind = "Private"
				}
				meta := fmt.Sprintf("%s, %d records", kind, aws.ToInt64(z.ResourceRecordSetCount))
				items = append(items, ui.Item{
					Name:      name,
					Meta:      meta,
					CopyValue: id,
					Child: func() *ui.Screen {
						return recordSetsScreen(c, name, id)
					},
				})
			}
			return items, nil
		},
	}
}

func recordSetsScreen(c *Client, zoneName, zoneID string) *ui.Screen {
	return &ui.Screen{
		Title: zoneName,
		Noun:  "records",
		List: func(ctx context.Context) ([]ui.Item, error) {
			records, err := c.ListResourceRecordSets(ctx, zoneID)
			if err != nil {
				return nil, err
			}
			items := make([]ui.Item, 0, len(records))
			for _, r := range records {
				items = append(items, recordItem(r))
			}
			return items, nil
		},
	}
}

func recordItem(r types.ResourceRecordSet) ui.Item {
	name := aws.ToString(r.Name)
	rtype := string(r.Type)
	meta := rtype
	if r.SetIdentifier != nil {
		meta += ", " + aws.ToString(r.SetIdentifier)
	}
	return ui.Item{
		Name:       name,
		Meta:       meta,
		Fields:     recordFields(r),
		ValueLabel: "Values",
		Value: func(ctx context.Context) (string, error) {
			return formatRecordValues(r), nil
		},
	}
}

func recordFields(r types.ResourceRecordSet) []ui.Field {
	fields := []ui.Field{
		{Label: "Name", Value: aws.ToString(r.Name)},
		{Label: "Type", Value: string(r.Type)},
	}
	if r.TTL != nil {
		fields = append(fields, ui.Field{Label: "TTL", Value: strconv.FormatInt(*r.TTL, 10)})
	}
	if r.SetIdentifier != nil {
		fields = append(fields, ui.Field{Label: "Set Identifier", Value: aws.ToString(r.SetIdentifier)})
	}
	if r.Weight != nil {
		fields = append(fields, ui.Field{Label: "Weight", Value: strconv.FormatInt(*r.Weight, 10)})
	}
	if r.Region != "" {
		fields = append(fields, ui.Field{Label: "Region", Value: string(r.Region)})
	}
	if r.Failover != "" {
		fields = append(fields, ui.Field{Label: "Failover", Value: string(r.Failover)})
	}
	if r.HealthCheckId != nil {
		fields = append(fields, ui.Field{Label: "Health Check", Value: aws.ToString(r.HealthCheckId)})
	}
	return fields
}

func formatRecordValues(r types.ResourceRecordSet) string {
	if r.AliasTarget != nil {
		health := "no"
		if r.AliasTarget.EvaluateTargetHealth {
			health = "yes"
		}
		return fmt.Sprintf("Alias -> %s\nHosted Zone: %s\nEvaluate Target Health: %s",
			aws.ToString(r.AliasTarget.DNSName), aws.ToString(r.AliasTarget.HostedZoneId), health)
	}
	values := make([]string, 0, len(r.ResourceRecords))
	for _, rr := range r.ResourceRecords {
		values = append(values, aws.ToString(rr.Value))
	}
	if len(values) == 0 {
		return "(no values)"
	}
	return strings.Join(values, "\n")
}

// shortZoneID strips the "/hostedzone/" prefix Route 53 puts on hosted zone
// IDs in list responses.
func shortZoneID(id string) string {
	if _, after, ok := strings.Cut(id, "/hostedzone/"); ok {
		return after
	}
	return id
}
