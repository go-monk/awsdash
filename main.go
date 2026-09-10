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

	resources, err := getResources(ctx, cfg, tags)
	if err != nil {
		log.Fatal(err)
	}

	dashboard.NamePrefix = *prefix

	if err := dashboard.Put(ctx, cfg, tags,
		widget.Text(dashboard.Header(tags), 24, 2),

		widget.Text("## Amplify Apps", 24, 1),
		widget.Metric(resources.Apps.Requests(cfg.Region), 8, 5),
		widget.Metric(resources.Apps.Errors4xx(cfg.Region), 8, 5),
		widget.Metric(resources.Apps.Errors5xx(cfg.Region), 8, 5),

		widget.Text("## API Gateways", 24, 1),
		widget.Metric(resources.APIs.Requests(cfg.Region), 8, 5),
		widget.Metric(resources.APIs.Errors4xx(cfg.Region), 8, 5),
		widget.Metric(resources.APIs.Errors5xx(cfg.Region), 8, 5),

		widget.Text("## Lambdas", 24, 1),
		widget.Metric(resources.Lambdas.Invocations(cfg.Region).LegendRight(), 12, 5),
		widget.Metric(resources.Lambdas.Errors(cfg.Region).LegendRight(), 12, 5),
		// widget.Metric(resources.Lambdas.ResponseErrors(cfg.Region), 8, 5),

		widget.Text("## S3 Buckets", 24, 1),
		widget.Metric(resources.Buckets.Size(cfg.Region).ShowUnits().LegendRight(), 12, 5),
		widget.Metric(resources.Buckets.Objects(cfg.Region).ShowUnits().LegendRight(), 12, 5),
	); err != nil {
		log.Fatal(err)
	}
}

// resources holds the AWS resources a dashboard is built from.
type resources struct {
	Apps    amplify.Apps
	APIs    apigateway.RESTs
	Lambdas lambda.Lambdas
	Buckets s3.Buckets
}

// getResources concurrently retrieves selected AWS resources matching tags. It
// returns the first error encountered, if any.
func getResources(ctx context.Context, cfg aws.Config, tags resource.Tags) (resources, error) {
	var r resources

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		r.Apps, err = amplify.Get(gctx, cfg, tags)
		log.Printf("found %d amplify apps\n", len(r.Apps))
		return err
	})
	g.Go(func() (err error) {
		r.APIs, err = apigateway.Get(gctx, cfg, tags)
		log.Printf("found %d api gateways\n", len(r.APIs))
		return err
	})
	g.Go(func() (err error) {
		r.Lambdas, err = lambda.Get(gctx, cfg, tags)
		log.Printf("found %d lambdas\n", len(r.Lambdas))
		return err
	})
	g.Go(func() (err error) {
		r.Buckets, err = s3.Get(gctx, cfg, tags)
		log.Printf("found %d s3 buckets\n", len(r.Buckets))
		return err
	})
	return r, g.Wait()
}
