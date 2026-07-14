package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/go-monk/awsdash/dashboard/widget"
)

func TestExplainSelectors(t *testing.T) {
	var selectors explainSelectors
	if err := selectors.Set("lambda.errors, s3.size"); err != nil {
		t.Fatalf("Set comma-separated selectors: %v", err)
	}
	if err := selectors.Set("apigateway.requests"); err != nil {
		t.Fatalf("Set repeated selector: %v", err)
	}

	want := explainSelectors{"lambda.errors", "s3.size", "apigateway.requests"}
	if !reflect.DeepEqual(selectors, want) {
		t.Fatalf("selectors = %v, want %v", selectors, want)
	}
}

func TestFormatExplainModelFlagUsage(t *testing.T) {
	tests := []struct {
		name      string
		installed []string
		err       error
		want      string
	}{
		{
			name:      "installed models",
			installed: []string{"vision-a", "vision-b"},
			want:      "(installed: vision-a, vision-b)",
		},
		{
			name: "no installed models",
			want: "no validated vision models installed",
		},
		{
			name: "discovery error",
			err:  errors.New("broken index"),
			want: "unable to inspect installed models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatExplainModelFlagUsage(tt.installed, tt.err)
			if !strings.Contains(got, tt.want) {
				t.Fatalf("usage = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

func TestBuildWidgetsUsesStableMetricIDs(t *testing.T) {
	items := buildWidgets("eu-central-1", nil, nil, nil, nil, nil)
	got := make([]string, 0, len(metricWidgetIDs))
	for _, item := range items {
		if item.ID != "" {
			got = append(got, item.ID)
		}
	}
	if !reflect.DeepEqual(got, metricWidgetIDs) {
		t.Fatalf("metric ids = %v, want %v", got, metricWidgetIDs)
	}
}

func TestSelectMetricWidgetsPreservesDashboardOrderAndDeduplicates(t *testing.T) {
	items := []dashboardWidget{
		{ID: "first", Widget: widget.Metric(widget.Properties{}, 1, 1)},
		{ID: "second", Widget: widget.Metric(widget.Properties{}, 1, 1)},
		{ID: "third", Widget: widget.Metric(widget.Properties{}, 1, 1)},
	}

	selected, err := selectMetricWidgets(items, []string{"third", "first", "third"})
	if err != nil {
		t.Fatalf("selectMetricWidgets: %v", err)
	}
	got := []string{selected[0].ID, selected[1].ID}
	want := []string{"first", "third"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selected ids = %v, want %v", got, want)
	}
}

func TestSelectMetricWidgetsAll(t *testing.T) {
	items := []dashboardWidget{
		{Widget: widget.Text("section", 1, 1)},
		{ID: "first", Widget: widget.Metric(widget.Properties{}, 1, 1)},
		{ID: "second", Widget: widget.Metric(widget.Properties{}, 1, 1)},
	}

	selected, err := selectMetricWidgets(items, []string{"all"})
	if err != nil {
		t.Fatalf("selectMetricWidgets: %v", err)
	}
	if got := len(selected); got != 2 {
		t.Fatalf("selected count = %d, want 2", got)
	}
}

func TestSelectMetricWidgetsRejectsUnknownID(t *testing.T) {
	items := []dashboardWidget{
		{ID: "lambda.errors", Widget: widget.Metric(widget.Properties{}, 1, 1)},
	}

	_, err := selectMetricWidgets(items, []string{"lambda.duration"})
	if err == nil {
		t.Fatal("selectMetricWidgets returned nil error for unknown id")
	}
	if !strings.Contains(err.Error(), "lambda.errors") {
		t.Fatalf("error %q does not list valid widget ids", err)
	}
}

func TestValidateExplainSelectors(t *testing.T) {
	if err := validateExplainSelectors([]string{"all", "lambda.errors"}); err != nil {
		t.Fatalf("validateExplainSelectors valid ids: %v", err)
	}

	err := validateExplainSelectors([]string{"lambda.duration"})
	if err == nil {
		t.Fatal("validateExplainSelectors returned nil error for unknown id")
	}
	if !strings.Contains(err.Error(), "lambda.errors") {
		t.Fatalf("error %q does not list valid widget ids", err)
	}
}

func TestResourcesForSelectors(t *testing.T) {
	tests := []struct {
		name      string
		selectors []string
		want      resourceSelection
	}{
		{
			name: "dashboard uses all resources",
			want: resourceSelection{amplify: true, apigateway: true, lambda: true, s3: true},
		},
		{
			name:      "single family",
			selectors: []string{"amplify.errors4xx"},
			want:      resourceSelection{amplify: true},
		},
		{
			name:      "multiple families",
			selectors: []string{"lambda.errors", "s3.size"},
			want:      resourceSelection{lambda: true, s3: true},
		},
		{
			name:      "all selector",
			selectors: []string{"all"},
			want:      resourceSelection{amplify: true, apigateway: true, lambda: true, s3: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resourcesForSelectors(tt.selectors); got != tt.want {
				t.Fatalf("resourcesForSelectors(%v) = %+v, want %+v", tt.selectors, got, tt.want)
			}
		})
	}
}
