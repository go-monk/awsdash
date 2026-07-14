package dashboard

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-monk/awsdash/dashboard/widget"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

type imageClient struct {
	input *cloudwatch.GetMetricWidgetImageInput
}

func (client *imageClient) GetMetricWidgetImage(
	_ context.Context,
	input *cloudwatch.GetMetricWidgetImageInput,
	_ ...func(*cloudwatch.Options),
) (*cloudwatch.GetMetricWidgetImageOutput, error) {
	client.input = input
	return &cloudwatch.GetMetricWidgetImageOutput{MetricWidgetImage: []byte("png")}, nil
}

func TestMetricWidgetImage(t *testing.T) {
	start := time.Date(2026, time.July, 14, 8, 0, 0, 0, time.FixedZone("test", 2*60*60))
	end := start.Add(6 * time.Hour)
	properties := widget.NewProperties("Lambda Errors", "eu-central-1", [][]any{
		{"AWS/Lambda", "Errors", "FunctionName", "checkout"},
	})
	client := &imageClient{}

	image, err := MetricWidgetImage(context.Background(), client, properties, start, end)
	if err != nil {
		t.Fatalf("MetricWidgetImage: %v", err)
	}
	if string(image) != "png" {
		t.Fatalf("image = %q, want png", image)
	}
	if got := *client.input.OutputFormat; got != "png" {
		t.Fatalf("output format = %q, want png", got)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(*client.input.MetricWidget), &payload); err != nil {
		t.Fatalf("unmarshal widget payload: %v", err)
	}

	wants := map[string]any{
		"title":  "Lambda Errors",
		"region": "eu-central-1",
		"start":  "2026-07-14T06:00:00Z",
		"end":    "2026-07-14T12:00:00Z",
		"width":  float64(metricImageWidth),
		"height": float64(metricImageHeight),
	}
	for key, want := range wants {
		if got := payload[key]; got != want {
			t.Errorf("payload[%q] = %#v, want %#v", key, got, want)
		}
	}
}

func TestMetricWidgetImageRejectsInvalidRange(t *testing.T) {
	now := time.Now()
	_, err := MetricWidgetImage(context.Background(), &imageClient{}, widget.Properties{}, now, now)
	if err == nil {
		t.Fatal("MetricWidgetImage returned nil error for an empty range")
	}
}
