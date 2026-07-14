package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-monk/awsdash/dashboard/widget"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

const (
	metricImageWidth  = 1200
	metricImageHeight = 600
)

// MetricWidgetImageAPI is the CloudWatch operation used to render a metric
// widget. It is satisfied by [cloudwatch.Client].
type MetricWidgetImageAPI interface {
	GetMetricWidgetImage(
		context.Context,
		*cloudwatch.GetMetricWidgetImageInput,
		...func(*cloudwatch.Options),
	) (*cloudwatch.GetMetricWidgetImageOutput, error)
}

// MetricWidgetImage renders metric properties over the requested time range as
// a PNG suitable for visual analysis.
func MetricWidgetImage(
	ctx context.Context,
	client MetricWidgetImageAPI,
	properties widget.Properties,
	start time.Time,
	end time.Time,
) ([]byte, error) {
	if !start.Before(end) {
		return nil, errors.New("metric image start must be before end")
	}

	payload, err := json.Marshal(struct {
		widget.Properties
		Start  string `json:"start"`
		End    string `json:"end"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}{
		Properties: properties,
		Start:      start.UTC().Format(time.RFC3339),
		End:        end.UTC().Format(time.RFC3339),
		Width:      metricImageWidth,
		Height:     metricImageHeight,
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling metric widget: %w", err)
	}

	output, err := client.GetMetricWidgetImage(ctx, &cloudwatch.GetMetricWidgetImageInput{
		MetricWidget: aws.String(string(payload)),
		OutputFormat: aws.String("png"),
	})
	if err != nil {
		return nil, fmt.Errorf("getting metric widget image: %w", err)
	}
	if output == nil || len(output.MetricWidgetImage) == 0 {
		return nil, errors.New("CloudWatch returned an empty metric widget image")
	}

	return output.MetricWidgetImage, nil
}
