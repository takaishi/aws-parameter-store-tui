package ecs

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"github.com/takaishi/aws-tui/internal/ui"
)

func clustersScreen(c *Client, region string) *ui.Screen {
	return &ui.Screen{
		Title: fmt.Sprintf("Amazon ECS (%s)", region),
		Noun:  "clusters",
		List: func(ctx context.Context) ([]ui.Item, error) {
			clusters, err := c.ListClusters(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]ui.Item, 0, len(clusters))
			for _, cl := range clusters {
				name := aws.ToString(cl.ClusterName)
				meta := fmt.Sprintf("%d services, %d tasks", cl.ActiveServicesCount, cl.RunningTasksCount)
				if status := aws.ToString(cl.Status); status != "ACTIVE" {
					meta = status + ", " + meta
				}
				items = append(items, ui.Item{
					Name:      name,
					Meta:      meta,
					CopyValue: aws.ToString(cl.ClusterArn),
					Child: func() *ui.Screen {
						return servicesScreen(c, name, name)
					},
				})
			}
			return items, nil
		},
	}
}

func servicesScreen(c *Client, title, cluster string) *ui.Screen {
	return &ui.Screen{
		Title: title,
		Noun:  "services",
		List: func(ctx context.Context) ([]ui.Item, error) {
			services, err := c.ListServices(ctx, cluster)
			if err != nil {
				return nil, err
			}
			items := make([]ui.Item, 0, len(services))
			for _, svc := range services {
				name := aws.ToString(svc.ServiceName)
				meta := fmt.Sprintf("%d/%d running", svc.RunningCount, svc.DesiredCount)
				if d := primaryDeployment(svc); d != nil && d.RolloutState != "" {
					meta += ", " + string(d.RolloutState)
				}
				items = append(items, ui.Item{
					Name:      name,
					Meta:      meta,
					CopyValue: aws.ToString(svc.ServiceArn),
					Child: func() *ui.Screen {
						return serviceScreen(c, cluster, name)
					},
				})
			}
			return items, nil
		},
	}
}

// serviceScreen lists a service's tasks, preceded by entries for the
// service's detail/events and its task definition, so everything fits into
// one pane in the columns layout.
func serviceScreen(c *Client, cluster, service string) *ui.Screen {
	return &ui.Screen{
		Title: service,
		Noun:  "tasks",
		List: func(ctx context.Context) ([]ui.Item, error) {
			svc, err := c.DescribeService(ctx, cluster, service)
			if err != nil {
				return nil, err
			}
			taskDefARN := aws.ToString(svc.TaskDefinition)
			items := []ui.Item{
				{
					Name:       "Detail & events",
					Fields:     serviceFields(svc),
					ValueLabel: "Events",
					Value: func(ctx context.Context) (string, error) {
						return formatEvents(svc.Events), nil
					},
				},
				{
					Name:      "Task definition",
					Meta:      shortTaskDef(taskDefARN),
					CopyValue: taskDefARN,
					Fields: []ui.Field{
						{Label: "Task Definition", Value: shortTaskDef(taskDefARN)},
						{Label: "ARN", Value: taskDefARN},
					},
					ValueLabel: "Definition",
					Value: func(ctx context.Context) (string, error) {
						td, err := c.DescribeTaskDefinition(ctx, taskDefARN)
						if err != nil {
							return "", err
						}
						return formatTaskDefinition(td), nil
					},
				},
			}
			tasks, err := c.ListServiceTasks(ctx, cluster, service)
			if err != nil {
				return nil, err
			}
			for _, t := range tasks {
				items = append(items, taskItem(t))
			}
			return items, nil
		},
	}
}

func taskItem(t types.Task) ui.Item {
	id := taskID(aws.ToString(t.TaskArn))
	status := aws.ToString(t.LastStatus)
	meta := status
	if t.HealthStatus != types.HealthStatusUnknown {
		meta += " (" + string(t.HealthStatus) + ")"
	}
	if reason := aws.ToString(t.StoppedReason); status == "STOPPED" && reason != "" {
		meta += " — " + truncateString(reason, 60)
	}
	return ui.Item{
		Name:       id,
		Meta:       meta,
		CopyValue:  aws.ToString(t.TaskArn),
		Fields:     taskFields(t, id),
		ValueLabel: "Containers",
		Value: func(ctx context.Context) (string, error) {
			return formatContainers(t.Containers), nil
		},
	}
}

func serviceFields(svc types.Service) []ui.Field {
	fields := []ui.Field{
		{Label: "Name", Value: aws.ToString(svc.ServiceName)},
		{Label: "ARN", Value: aws.ToString(svc.ServiceArn)},
		{Label: "Status", Value: aws.ToString(svc.Status)},
		{Label: "Task Definition", Value: shortTaskDef(aws.ToString(svc.TaskDefinition))},
		{Label: "Desired / Running / Pending", Value: fmt.Sprintf("%d / %d / %d", svc.DesiredCount, svc.RunningCount, svc.PendingCount)},
	}
	if svc.LaunchType != "" {
		fields = append(fields, ui.Field{Label: "Launch Type", Value: string(svc.LaunchType)})
	}
	if d := primaryDeployment(svc); d != nil {
		if d.RolloutState != "" {
			fields = append(fields, ui.Field{Label: "Rollout", Value: string(d.RolloutState)})
		}
		if reason := aws.ToString(d.RolloutStateReason); reason != "" {
			fields = append(fields, ui.Field{Label: "Rollout Reason", Value: reason})
		}
	}
	return fields
}

func taskFields(t types.Task, id string) []ui.Field {
	fields := []ui.Field{
		{Label: "Task ID", Value: id},
		{Label: "ARN", Value: aws.ToString(t.TaskArn)},
		{Label: "Task Definition", Value: shortTaskDef(aws.ToString(t.TaskDefinitionArn))},
		{Label: "Status", Value: fmt.Sprintf("%s (desired %s)", aws.ToString(t.LastStatus), aws.ToString(t.DesiredStatus))},
	}
	if t.HealthStatus != types.HealthStatusUnknown {
		fields = append(fields, ui.Field{Label: "Health", Value: string(t.HealthStatus)})
	}
	if t.LaunchType != "" {
		fields = append(fields, ui.Field{Label: "Launch Type", Value: string(t.LaunchType)})
	}
	if az := aws.ToString(t.AvailabilityZone); az != "" {
		fields = append(fields, ui.Field{Label: "AZ", Value: az})
	}
	if ip := taskPrivateIP(t); ip != "" {
		fields = append(fields, ui.Field{Label: "Private IP", Value: ip})
	}
	if cpu, mem := aws.ToString(t.Cpu), aws.ToString(t.Memory); cpu != "" || mem != "" {
		fields = append(fields, ui.Field{Label: "CPU / Memory", Value: cpu + " / " + mem})
	}
	if t.StartedAt != nil {
		fields = append(fields, ui.Field{Label: "Started At", Value: fmtTime(t.StartedAt)})
	}
	if by := aws.ToString(t.StartedBy); by != "" {
		fields = append(fields, ui.Field{Label: "Started By", Value: by})
	}
	if t.StoppedAt != nil {
		fields = append(fields, ui.Field{Label: "Stopped At", Value: fmtTime(t.StoppedAt)})
	}
	if t.StopCode != "" {
		fields = append(fields, ui.Field{Label: "Stop Code", Value: string(t.StopCode)})
	}
	if reason := aws.ToString(t.StoppedReason); reason != "" {
		fields = append(fields, ui.Field{Label: "Stopped Reason", Value: reason})
	}
	return fields
}

func formatEvents(events []types.ServiceEvent) string {
	if len(events) == 0 {
		return "(no events)"
	}
	// Events come newest first; show the most recent 50.
	n := len(events)
	if n > 50 {
		n = 50
	}
	var b strings.Builder
	for _, e := range events[:n] {
		fmt.Fprintf(&b, "%s  %s\n", fmtTime(e.CreatedAt), aws.ToString(e.Message))
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatContainers(containers []types.Container) string {
	if len(containers) == 0 {
		return "(no containers)"
	}
	var b strings.Builder
	for i, c := range containers {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "[%s]\n", aws.ToString(c.Name))
		fmt.Fprintf(&b, "  Image:  %s\n", aws.ToString(c.Image))
		status := aws.ToString(c.LastStatus)
		if c.HealthStatus != types.HealthStatusUnknown {
			status += " (" + string(c.HealthStatus) + ")"
		}
		fmt.Fprintf(&b, "  Status: %s\n", status)
		if c.ExitCode != nil {
			fmt.Fprintf(&b, "  Exit Code: %d\n", *c.ExitCode)
		}
		if reason := aws.ToString(c.Reason); reason != "" {
			fmt.Fprintf(&b, "  Reason: %s\n", reason)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatTaskDefinition(td *types.TaskDefinition) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Family:       %s:%d\n", aws.ToString(td.Family), td.Revision)
	if cpu, mem := aws.ToString(td.Cpu), aws.ToString(td.Memory); cpu != "" || mem != "" {
		fmt.Fprintf(&b, "CPU / Memory: %s / %s\n", cpu, mem)
	}
	if td.NetworkMode != "" {
		fmt.Fprintf(&b, "Network Mode: %s\n", td.NetworkMode)
	}
	for _, cd := range td.ContainerDefinitions {
		fmt.Fprintf(&b, "\n[%s]\n", aws.ToString(cd.Name))
		fmt.Fprintf(&b, "Image: %s\n", aws.ToString(cd.Image))
		if cd.Cpu != 0 || cd.Memory != nil || cd.MemoryReservation != nil {
			parts := []string{}
			if cd.Cpu != 0 {
				parts = append(parts, "cpu="+strconv.FormatInt(int64(cd.Cpu), 10))
			}
			if cd.Memory != nil {
				parts = append(parts, "memory="+strconv.FormatInt(int64(*cd.Memory), 10))
			}
			if cd.MemoryReservation != nil {
				parts = append(parts, "memoryReservation="+strconv.FormatInt(int64(*cd.MemoryReservation), 10))
			}
			fmt.Fprintf(&b, "Resources: %s\n", strings.Join(parts, " "))
		}
		if cd.Essential != nil && !*cd.Essential {
			b.WriteString("Essential: false\n")
		}
		if len(cd.PortMappings) > 0 {
			ports := make([]string, 0, len(cd.PortMappings))
			for _, pm := range cd.PortMappings {
				port := strconv.FormatInt(int64(aws.ToInt32(pm.ContainerPort)), 10)
				if pm.Protocol != "" {
					port += "/" + string(pm.Protocol)
				}
				ports = append(ports, port)
			}
			fmt.Fprintf(&b, "Ports: %s\n", strings.Join(ports, ", "))
		}
		if len(cd.Environment) > 0 {
			b.WriteString("Environment:\n")
			for _, kv := range cd.Environment {
				fmt.Fprintf(&b, "  %s=%s\n", aws.ToString(kv.Name), aws.ToString(kv.Value))
			}
		}
		if len(cd.Secrets) > 0 {
			b.WriteString("Secrets:\n")
			for _, s := range cd.Secrets {
				fmt.Fprintf(&b, "  %s ← %s\n", aws.ToString(s.Name), aws.ToString(s.ValueFrom))
			}
		}
		if lc := cd.LogConfiguration; lc != nil {
			fmt.Fprintf(&b, "Log Driver: %s\n", lc.LogDriver)
			if len(lc.Options) > 0 {
				keys := make([]string, 0, len(lc.Options))
				for k := range lc.Options {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Fprintf(&b, "  %s=%s\n", k, lc.Options[k])
				}
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func primaryDeployment(svc types.Service) *types.Deployment {
	for i := range svc.Deployments {
		if aws.ToString(svc.Deployments[i].Status) == "PRIMARY" {
			return &svc.Deployments[i]
		}
	}
	return nil
}

func taskPrivateIP(t types.Task) string {
	for _, c := range t.Containers {
		for _, ni := range c.NetworkInterfaces {
			if ip := aws.ToString(ni.PrivateIpv4Address); ip != "" {
				return ip
			}
		}
	}
	for _, a := range t.Attachments {
		for _, d := range a.Details {
			if aws.ToString(d.Name) == "privateIPv4Address" {
				return aws.ToString(d.Value)
			}
		}
	}
	return ""
}

// shortTaskDef turns a task definition ARN into "family:revision".
func shortTaskDef(arn string) string {
	if _, after, ok := strings.Cut(arn, "task-definition/"); ok {
		return after
	}
	return arn
}

// taskID returns the trailing ID segment of a task ARN.
func taskID(arn string) string {
	if i := strings.LastIndex(arn, "/"); i >= 0 {
		return arn[i+1:]
	}
	return arn
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func truncateString(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
