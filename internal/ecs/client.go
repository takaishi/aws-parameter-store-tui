package ecs

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type Client struct {
	client *awsecs.Client
}

func NewClient(cfg aws.Config) *Client {
	return &Client{client: awsecs.NewFromConfig(cfg)}
}

func (c *Client) ListClusters(ctx context.Context) ([]types.Cluster, error) {
	var arns []string
	paginator := awsecs.NewListClustersPaginator(c.client, &awsecs.ListClustersInput{})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		arns = append(arns, out.ClusterArns...)
	}
	var clusters []types.Cluster
	for _, batch := range chunk(arns, 100) {
		out, err := c.client.DescribeClusters(ctx, &awsecs.DescribeClustersInput{
			Clusters: batch,
		})
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, out.Clusters...)
	}
	sort.Slice(clusters, func(i, j int) bool {
		return aws.ToString(clusters[i].ClusterName) < aws.ToString(clusters[j].ClusterName)
	})
	return clusters, nil
}

func (c *Client) ListServices(ctx context.Context, cluster string) ([]types.Service, error) {
	var arns []string
	paginator := awsecs.NewListServicesPaginator(c.client, &awsecs.ListServicesInput{
		Cluster: aws.String(cluster),
	})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		arns = append(arns, out.ServiceArns...)
	}
	var services []types.Service
	// DescribeServices accepts at most 10 services per call.
	for _, batch := range chunk(arns, 10) {
		out, err := c.client.DescribeServices(ctx, &awsecs.DescribeServicesInput{
			Cluster:  aws.String(cluster),
			Services: batch,
		})
		if err != nil {
			return nil, err
		}
		services = append(services, out.Services...)
	}
	sort.Slice(services, func(i, j int) bool {
		return aws.ToString(services[i].ServiceName) < aws.ToString(services[j].ServiceName)
	})
	return services, nil
}

func (c *Client) DescribeService(ctx context.Context, cluster, service string) (types.Service, error) {
	out, err := c.client.DescribeServices(ctx, &awsecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: []string{service},
	})
	if err != nil {
		return types.Service{}, err
	}
	if len(out.Services) == 0 {
		return types.Service{}, fmt.Errorf("service %s not found in cluster %s", service, cluster)
	}
	return out.Services[0], nil
}

// ListServiceTasks returns the service's running and recently stopped tasks:
// running tasks first, newest first within each group.
func (c *Client) ListServiceTasks(ctx context.Context, cluster, service string) ([]types.Task, error) {
	var arns []string
	for _, status := range []types.DesiredStatus{types.DesiredStatusRunning, types.DesiredStatusStopped} {
		paginator := awsecs.NewListTasksPaginator(c.client, &awsecs.ListTasksInput{
			Cluster:       aws.String(cluster),
			ServiceName:   aws.String(service),
			DesiredStatus: status,
		})
		for paginator.HasMorePages() {
			out, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			arns = append(arns, out.TaskArns...)
		}
	}
	var tasks []types.Task
	for _, batch := range chunk(arns, 100) {
		out, err := c.client.DescribeTasks(ctx, &awsecs.DescribeTasksInput{
			Cluster: aws.String(cluster),
			Tasks:   batch,
		})
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, out.Tasks...)
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		ri := aws.ToString(tasks[i].LastStatus) != "STOPPED"
		rj := aws.ToString(tasks[j].LastStatus) != "STOPPED"
		if ri != rj {
			return ri
		}
		ti, tj := tasks[i].StartedAt, tasks[j].StartedAt
		if ti == nil || tj == nil {
			return tj == nil && ti != nil
		}
		return ti.After(*tj)
	})
	return tasks, nil
}

func (c *Client) DescribeTaskDefinition(ctx context.Context, taskDefinition string) (*types.TaskDefinition, error) {
	out, err := c.client.DescribeTaskDefinition(ctx, &awsecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefinition),
	})
	if err != nil {
		return nil, err
	}
	return out.TaskDefinition, nil
}

func chunk(items []string, size int) [][]string {
	var chunks [][]string
	for len(items) > 0 {
		n := size
		if len(items) < n {
			n = len(items)
		}
		chunks = append(chunks, items[:n])
		items = items[n:]
	}
	return chunks
}
