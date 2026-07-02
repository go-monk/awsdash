package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-monk/awsdash/dashboard"
	"github.com/go-monk/awsdash/dashboard/widget"
	"github.com/go-monk/awsdash/resource"
	"github.com/go-monk/awsdash/resource/amplify"
	"github.com/go-monk/awsdash/resource/apigateway"
	"github.com/go-monk/awsdash/resource/lambda"
	"github.com/go-monk/awsdash/resource/s3"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"golang.org/x/sync/errgroup"
)

func init() {
	log.SetPrefix(filepath.Base(os.Args[0]) + ": ")
	log.SetFlags(0)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Awsdash creates or updates custom AWS CloudWatch dashboards.\n")
		flag.PrintDefaults()
	}
	var tags resource.Tags
	flag.Var(&tags, "tags", "resource tags as key=value pairs; comma-seperated or repeated")
	prefix := flag.String("prefix", dashboard.NamePrefix, "dashboard name prefix")
	flag.Parse()

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("loading AWS config: %v", err)
	}

	apps, apis, lambdas, buckets, err := getResources(ctx, cfg, tags)
	if err != nil {
		log.Fatal(err)
	}

	dashboard.NamePrefix = *prefix

	if err := dashboard.Put(ctx, cfg, tags,
		widget.Text(dashboard.Header(tags), 24, 2),

		widget.Text("## Amplify Apps", 24, 1),
		widget.Metric(apps.Requests(cfg.Region), 8, 5),
		widget.Metric(apps.Errors4xx(cfg.Region), 8, 5),
		widget.Metric(apps.Errors5xx(cfg.Region), 8, 5),

		widget.Text("## API Gateways", 24, 1),
		widget.Metric(apis.Requests(cfg.Region), 8, 5),
		widget.Metric(apis.Errors4xx(cfg.Region), 8, 5),
		widget.Metric(apis.Errors5xx(cfg.Region), 8, 5),

		widget.Text("## Lambdas", 24, 1),
		widget.Metric(lambdas.Invocations(cfg.Region).LegendRight(), 12, 5),
		widget.Metric(lambdas.Errors(cfg.Region).LegendRight(), 12, 5),
		// widget.Metric(lambdas.ResponseErrors(cfg.Region), 8, 5),

		widget.Text("## S3 Buckets", 24, 1),
		widget.Metric(buckets.Size(cfg.Region).ShowUnits().LegendRight(), 12, 5),
		widget.Metric(buckets.Objects(cfg.Region).ShowUnits().LegendRight(), 12, 5),
	); err != nil {
		log.Fatal(err)
	}
}

// getResources concurrently retrieves selected AWS resources matching tags. It
// returns the first error encountered, if any.
func getResources(ctx context.Context, cfg aws.Config, tags resource.Tags) (amplify.Apps, apigateway.RESTs, lambda.Lambdas, s3.Buckets, error) {
	var (
		apps    amplify.Apps
		apis    apigateway.RESTs
		lambdas lambda.Lambdas
		buckets s3.Buckets
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		apps, err = amplify.Get(gctx, cfg, tags)
		log.Printf("found %d amplify apps\n", len(apps))
		return err
	})
	g.Go(func() (err error) {
		apis, err = apigateway.Get(gctx, cfg, tags)
		log.Printf("found %d api gateways\n", len(apis))
		return err
	})
	g.Go(func() (err error) {
		lambdas, err = lambda.Get(gctx, cfg, tags)
		log.Printf("found %d lambdas\n", len(lambdas))
		return err
	})
	g.Go(func() (err error) {
		buckets, err = s3.Get(gctx, cfg, tags)
		log.Printf("found %d s3 buckets\n", len(buckets))
		return err
	})
	return apps, apis, lambdas, buckets, g.Wait()
}
