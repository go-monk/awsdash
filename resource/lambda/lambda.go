package lambda

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"

	"github.com/go-monk/awsdash/dashboard/widget"
	"github.com/go-monk/awsdash/resource"
)

type Lambda struct {
	Name string
}

type Lambdas []Lambda

func Get(ctx context.Context, cfg aws.Config, tags resource.Tags) (Lambdas, error) {
	arns, err := resource.GetARNs(ctx, cfg, []string{"lambda:function"}, tags)
	if err != nil {
		return nil, err
	}

	lambdas := make([]Lambda, 0, len(arns))
	for _, arn := range arns {
		name, err := extractFunctionName(arn)
		if err != nil {
			return nil, err
		}
		lambdas = append(lambdas, Lambda{Name: name})
	}
	return lambdas, nil
}

// extractFunctionName extracts the function name from a Lambda function ARN
// that has the following format: arn:aws:lambda:region:account:function:function-name
func extractFunctionName(arnStr string) (string, error) {
	parsedArn, err := arn.Parse(arnStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse arn: %w", err)
	}

	if parsedArn.Service != "lambda" {
		return "", fmt.Errorf("not lambda function arn: %s", arnStr)
	}

	// Resource format: "function:function-name" or "function:function-name:version"
	parts := strings.Split(parsedArn.Resource, ":")
	if len(parts) < 2 || parts[0] != "function" {
		return "", fmt.Errorf("invalid lambda function resource format: %s", arnStr)
	}

	return parts[1], nil
}

func (ll Lambdas) Invocations(region string) widget.Properties {
	return ll.metrics(region, "Invocations", "Invocations")
}

func (ll Lambdas) Errors(region string) widget.Properties {
	return ll.metrics(region, "Errors", "Errors")
}

func (ll Lambdas) ResponseErrors(region string) widget.Properties {
	return ll.metrics(region, "ResponseErrors", "Response Errors")
}

func (ll Lambdas) metrics(region, metricName, title string) widget.Properties {
	metrics := make([][]any, 0, len(ll))
	for i, l := range ll {
		color := widget.Colors[i%len(widget.Colors)]
		metrics = append(metrics, metric(l.Name, metricName, color, widget.StatSum))
	}
	return widget.NewProperties(title, region, metrics)
}

func metric(functionName, metric, color, stat string) []any {
	return []any{
		"AWS/Lambda", metric, "FunctionName", functionName,
		map[string]string{widget.KeyStat: stat, widget.KeyColor: color, widget.KeyLabel: functionName},
	}
}
