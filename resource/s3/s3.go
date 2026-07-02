package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"

	"github.com/go-monk/awsdash/dashboard/widget"
	"github.com/go-monk/awsdash/resource"
)

type Bucket struct {
	Name string
}

type Buckets []Bucket

func Get(ctx context.Context, cfg aws.Config, tags map[string]string) (Buckets, error) {
	arns, err := resource.GetARNs(ctx, cfg, []string{"s3"}, tags)
	if err != nil {
		return nil, err
	}

	buckets := make(Buckets, 0, len(arns))
	for _, arn := range arns {
		name, err := extractBucketName(arn)
		if err != nil {
			return nil, err
		}
		buckets = append(buckets, Bucket{Name: name})
	}

	return buckets, nil
}

func extractBucketName(arnStr string) (string, error) {
	// Bucket: arn:aws:s3:::bucket_name
	parsedArn, err := arn.Parse(arnStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse arn: %w", err)
	}

	if parsedArn.Service != "s3" {
		return "", fmt.Errorf("not s3 arn: %s", arnStr)
	}

	resource := parsedArn.Resource
	if resource == "" {
		return "", fmt.Errorf("empty bucket name in arn: %s", arnStr)
	}

	// Object: arn:aws:s3:::bucket_name/object_key
	if idx := strings.Index(resource, "/"); idx >= 0 {
		resource = resource[:idx]
	}

	return resource, nil
}

func (bb Buckets) Objects(region string) widget.Properties {
	return bb.metrics(region, "NumberOfObjects", "AllStorageTypes", "Number of Objects (average)")
}

func (bb Buckets) Size(region string) widget.Properties {
	return bb.metrics(region, "BucketSizeBytes", "StandardStorage", "Bucket Size (average)")
}

func (bb Buckets) metrics(region, metricName, storageType, widgetTitle string) widget.Properties {
	metrics := make([][]any, 0, len(bb))
	for i, b := range bb {
		color := widget.Colors[i%len(widget.Colors)]
		metrics = append(metrics, metric(b.Name, metricName, storageType, color))
	}
	return widget.NewProperties(widgetTitle, region, metrics)
}

func metric(bucketName, metric, storageType, color string) []any {
	return []any{
		"AWS/S3", metric, "BucketName", bucketName, "StorageType", storageType,
		map[string]any{
			"period":        86400,
			widget.KeyColor: color,
			widget.KeyLabel: bucketName,
			widget.KeyStat:  widget.StatAverage,
		},
	}
}
