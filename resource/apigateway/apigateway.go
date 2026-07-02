package apigateway

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"

	"github.com/go-monk/awsdash/dashboard/widget"
	"github.com/go-monk/awsdash/resource"
)

type REST struct {
	Name  string
	Stage string
}

type RESTs []REST

func Get(ctx context.Context, cfg aws.Config, tags map[string]string) (RESTs, error) {
	var rests RESTs

	client := apigateway.NewFromConfig(cfg)
	paginator := apigateway.NewGetRestApisPaginator(client, &apigateway.GetRestApisInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error listing REST APIs: %w", err)
		}

		for _, api := range page.Items {
			if len(tags) > 0 {
				if !resource.MatchesAllTags(api.Tags, tags) {
					continue
				}
			}

			stages, err := client.GetStages(ctx, &apigateway.GetStagesInput{
				RestApiId: api.Id,
			})
			if err != nil {
				return nil, fmt.Errorf("error listing stages for REST API %q: %w", aws.ToString(api.Name), err)
			}

			for _, stage := range stages.Item {
				rests = append(rests, REST{
					Name:  aws.ToString(api.Name),
					Stage: aws.ToString(stage.StageName),
				})
			}
		}
	}

	return rests, nil
}

func (rests RESTs) Requests(region string) widget.Properties {
	return rests.metrics(region, "Count", "Requests")
}

func (rests RESTs) Errors4xx(region string) widget.Properties {
	return rests.metrics(region, "4XXError", "4xx Errors")
}

func (rests RESTs) Errors5xx(region string) widget.Properties {
	return rests.metrics(region, "5XXError", "5xx Errors")
}

func (rests RESTs) metrics(region, metricName, title string) widget.Properties {
	metrics := make([][]any, 0, len(rests))
	for i, rest := range rests {
		color := widget.Colors[i%len(widget.Colors)]
		metrics = append(metrics, metric(rest.Name, rest.Stage, metricName, color, widget.StatSum))
	}
	return widget.NewProperties(title, region, metrics)
}

func metric(apiName, stage, metric, color, stat string) []any {
	label := fmt.Sprintf("%s (%s)", apiName, stage)
	return []any{
		"AWS/ApiGateway", metric, "ApiName", apiName, "Stage", stage,
		map[string]string{widget.KeyStat: stat, widget.KeyColor: color, widget.KeyLabel: label},
	}
}
