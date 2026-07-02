package amplify

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/amplify"

	"github.com/go-monk/awsdash/dashboard/widget"
	"github.com/go-monk/awsdash/resource"
)

type App struct {
	ID   string
	Name string
}

type Apps []App

func Get(ctx context.Context, cfg aws.Config, tags map[string]string) (Apps, error) {
	var apps []App

	client := amplify.NewFromConfig(cfg)
	paginator := amplify.NewListAppsPaginator(client, &amplify.ListAppsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error listing apps: %w", err)
		}

		for _, app := range page.Apps {
			if len(tags) > 0 {
				if !resource.MatchesAllTags(app.Tags, tags) {
					continue
				}
			}

			apps = append(apps, App{
				ID:   *app.AppId,
				Name: *app.Name,
			})
		}
	}

	return apps, nil
}

func (apps Apps) Requests(region string) widget.Properties {
	return apps.metrics(region, "Requests", "Requests")
}

func (apps Apps) Errors4xx(region string) widget.Properties {
	return apps.metrics(region, "4xxErrors", "4xx Errors")
}

func (apps Apps) Errors5xx(region string) widget.Properties {
	return apps.metrics(region, "5xxErrors", "5xx Errors")
}

func (apps Apps) metrics(region, metricName, title string) widget.Properties {
	metrics := make([][]any, 0, len(apps))
	for i, app := range apps {
		color := widget.Colors[i%len(widget.Colors)]
		metrics = append(metrics, metric(app.ID, app.Name, metricName, color, widget.StatSum))
	}
	return widget.NewProperties(title, region, metrics)
}

func metric(appId, appName, metric, color, stat string) []any {
	return []any{
		"AWS/AmplifyHosting", metric, "App", appId,
		map[string]string{widget.KeyStat: stat, widget.KeyColor: color, widget.KeyLabel: appName},
	}
}
