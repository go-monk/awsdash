package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/go-monk/awsdash/dashboard/widget"
	"github.com/go-monk/awsdash/resource"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

// NamePrefix is the dashboard name prefix.
var NamePrefix = "awsdash"

type Dashboard struct {
	Cfg  aws.Config
	Tags map[string]string
}

// Name derives a dashboard name from the [NamePrefix] and tags so that each tag set
// maps to a stable, unique dashboard.
func Name(tags map[string]string) string {
	if len(tags) == 0 {
		return NamePrefix
	}
	t := resource.Tags(tags)
	return strings.Join([]string{NamePrefix, t.String()}, "_")
}

// List returns matching dashboard names.
func List(ctx context.Context, cfg aws.Config, tags map[string]string) ([]string, error) {
	client := cloudwatch.NewFromConfig(cfg)

	names := make([]string, 0)
	var nextToken *string

	for {
		output, err := client.ListDashboards(ctx, &cloudwatch.ListDashboardsInput{
			DashboardNamePrefix: aws.String(Name(tags)),
			NextToken:           nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("listing dashboards by prefix %q: %w", Name(tags), err)
		}

		for _, entry := range output.DashboardEntries {
			if entry.DashboardName == nil {
				continue
			}
			names = append(names, *entry.DashboardName)
		}

		if output.NextToken == nil || *output.NextToken == "" {
			break
		}
		nextToken = output.NextToken
	}

	return names, nil
}

// Delete removes the dashboard. Deleting non-existent dahsboard does not return
// an error.
func Delete(ctx context.Context, cfg aws.Config, tags map[string]string) error {
	client := cloudwatch.NewFromConfig(cfg)
	_, err := client.DeleteDashboards(ctx, &cloudwatch.DeleteDashboardsInput{
		DashboardNames: []string{Name(tags)},
	})
	if err != nil {
		return fmt.Errorf("deleting dashboards %v: %w", Name(tags), err)
	}
	return nil
}

// Put creates or updates a dashboard containing the widgets.
func Put(ctx context.Context, cfg aws.Config, tags map[string]string, widgets ...widget.Widget) error {
	body, err := body(widgets...)
	if err != nil {
		return fmt.Errorf("generating dashboard body: %w", err)
	}

	client := cloudwatch.NewFromConfig(cfg)
	if _, err := client.PutDashboard(ctx, &cloudwatch.PutDashboardInput{
		DashboardName: aws.String(Name(tags)),
		DashboardBody: aws.String(body),
	}); err != nil {
		return err
	}

	return nil
}

func body(widgets ...widget.Widget) (string, error) {
	b, err := json.Marshal(struct {
		Widgets []widget.Widget `json:"widgets"`
	}{Widgets: widgets})

	if err != nil {
		return "", fmt.Errorf("marshaling dashboard JSON: %w", err)
	}

	return string(b), nil
}

// Header generates the markdown Header for the dashboard with tag information.
func Header(tags map[string]string) string {
	title := "# Generated with [awsdash](https://github.com/go-monk/awsdash)"
	if len(tags) == 0 {
		return title + "\nfor all resources"
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build tag list
	tagStrings := make([]string, len(keys))
	for i, key := range keys {
		tagStrings[i] = fmt.Sprintf("**%s**=%s", key, tags[key])
	}
	tagsList := strings.Join(tagStrings, ", ")

	return title + "\nfor resources with tags: " + tagsList
}
