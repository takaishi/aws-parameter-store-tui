package pstui

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type Parameter struct {
	Name         string
	Type         string
	Version      int64
	LastModified string
}

type SSMClient struct {
	client *ssm.Client
}

func NewSSMClient(cfg aws.Config) *SSMClient {
	return &SSMClient{client: ssm.NewFromConfig(cfg)}
}

func (c *SSMClient) ListParameters(ctx context.Context) ([]Parameter, error) {
	var params []Parameter
	paginator := ssm.NewDescribeParametersPaginator(c.client, &ssm.DescribeParametersInput{
		MaxResults: aws.Int32(50),
	})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, p := range out.Parameters {
			param := Parameter{
				Name:    aws.ToString(p.Name),
				Type:    string(p.Type),
				Version: p.Version,
			}
			if p.LastModifiedDate != nil {
				param.LastModified = p.LastModifiedDate.Local().Format("2006-01-02 15:04:05")
			}
			params = append(params, param)
		}
	}
	sort.Slice(params, func(i, j int) bool { return params[i].Name < params[j].Name })
	return params, nil
}

func (c *SSMClient) GetParameterValue(ctx context.Context, name string) (string, error) {
	out, err := c.client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.Parameter.Value), nil
}
