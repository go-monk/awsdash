package resource

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
)

func GetARNs(
	ctx context.Context,
	cfg aws.Config,
	resourceTypeFilters []string,
	tags map[string]string,
) ([]string, error) {
	client := resourcegroupstaggingapi.NewFromConfig(cfg)

	var ARNs []string

	var nextToken *string
	awsTagFilters := make([]types.TagFilter, 0, len(tags))
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := tags[key]
		awsTagFilters = append(awsTagFilters, types.TagFilter{
			Key:    aws.String(key),
			Values: []string{value},
		})
	}

	for {
		input := &resourcegroupstaggingapi.GetResourcesInput{
			ResourceTypeFilters: resourceTypeFilters,
			TagFilters:          awsTagFilters,
			PaginationToken:     nextToken,
		}

		output, err := client.GetResources(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get resources: %w", err)
		}

		// Extract function ARNs from response
		for _, resource := range output.ResourceTagMappingList {
			if resource.ResourceARN != nil {
				ARNs = append(ARNs, *resource.ResourceARN)
			}
		}

		// Check if there are more pages
		if output.PaginationToken == nil || *output.PaginationToken == "" {
			break
		}
		nextToken = output.PaginationToken
	}

	return ARNs, nil
}
