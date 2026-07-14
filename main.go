package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-monk/awsdash/dashboard"
	"github.com/go-monk/awsdash/dashboard/widget"
	"github.com/go-monk/awsdash/explain"
	"github.com/go-monk/awsdash/resource"
	"github.com/go-monk/awsdash/resource/amplify"
	"github.com/go-monk/awsdash/resource/apigateway"
	"github.com/go-monk/awsdash/resource/lambda"
	"github.com/go-monk/awsdash/resource/s3"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"golang.org/x/sync/errgroup"
)

const defaultExplainSince = 3 * time.Hour

const explainModelFlagDescription = "Kronk vision model id, provider/model id, or Hugging Face URL"

var metricWidgetIDs = []string{
	"amplify.requests",
	"amplify.errors4xx",
	"amplify.errors5xx",
	"apigateway.requests",
	"apigateway.errors4xx",
	"apigateway.errors5xx",
	"lambda.invocations",
	"lambda.errors",
	"s3.size",
	"s3.objects",
}

func init() {
	log.SetPrefix(filepath.Base(os.Args[0]) + ": ")
	log.SetFlags(0)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ContinueOnError)
	flags.Usage = func() {
		if modelFlag := flags.Lookup("explain-model"); modelFlag != nil {
			modelFlag.Usage = explainModelFlagUsage()
		}
		fmt.Fprintln(flags.Output(), "Awsdash creates, updates or explains custom AWS CloudWatch dashboards.")
		fmt.Fprintln(flags.Output())
		flags.PrintDefaults()
	}

	var (
		tags        resource.Tags
		selectors   explainSelectors
		validIDs    = strings.Join(metricWidgetIDs, ", ")
		prefix      = flags.String("prefix", dashboard.NamePrefix, "dashboard name prefix")
		since       = flags.Duration("explain-since", defaultExplainSince, "metric time range used for explanations")
		modelSource = flags.String("explain-model", "", explainModelFlagDescription)
		verbose     = flags.Bool("explain-verbose", false, "show Kronk model and inference logs")
	)
	flags.Var(&tags, "tags", "resource tags as key=value pairs; comma-separated or repeated")
	flags.Var(&selectors, "explain", "metric widget id; comma-separated or repeated; use all (valid: "+validIDs+")")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if len(selectors) > 0 {
		if strings.TrimSpace(*modelSource) == "" {
			return errors.New("-explain-model is required when -explain is used")
		}
		if *since <= 0 {
			return errors.New("-explain-since must be greater than zero")
		}
		if err := validateExplainSelectors(selectors); err != nil {
			return err
		}
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	apps, apis, lambdas, buckets, err := getResources(ctx, cfg, tags, resourcesForSelectors(selectors))
	if err != nil {
		return err
	}

	dashboard.NamePrefix = *prefix
	items := buildWidgets(cfg.Region, tags, apps, apis, lambdas, buckets)

	if len(selectors) > 0 {
		selected, err := selectMetricWidgets(items, selectors)
		if err != nil {
			return err
		}
		return explainWidgets(ctx, stdout, cfg, selected, *since, strings.TrimSpace(*modelSource), *verbose)
	}

	return dashboard.Put(ctx, cfg, tags, widgetValues(items)...)
}

func explainModelFlagUsage() string {
	installed, err := explain.InstalledVisionModels()
	return formatExplainModelFlagUsage(installed, err)
}

func formatExplainModelFlagUsage(installed []string, err error) string {
	switch {
	case err != nil:
		return explainModelFlagDescription + "; unable to inspect installed models (run: kronk model list --local)"
	case len(installed) == 0:
		return explainModelFlagDescription + "; no validated vision models installed (run: kronk catalog list --local)"
	default:
		return explainModelFlagDescription + " (installed: " + strings.Join(installed, ", ") + ")"
	}
}

type dashboardWidget struct {
	ID     string
	Widget widget.Widget
}

func buildWidgets(
	region string,
	tags resource.Tags,
	apps amplify.Apps,
	apis apigateway.RESTs,
	lambdas lambda.Lambdas,
	buckets s3.Buckets,
) []dashboardWidget {
	return []dashboardWidget{
		{Widget: widget.Text(dashboard.Header(tags), 24, 2)},

		{Widget: widget.Text("## Amplify Apps", 24, 1)},
		{ID: "amplify.requests", Widget: widget.Metric(apps.Requests(region), 8, 5)},
		{ID: "amplify.errors4xx", Widget: widget.Metric(apps.Errors4xx(region), 8, 5)},
		{ID: "amplify.errors5xx", Widget: widget.Metric(apps.Errors5xx(region), 8, 5)},

		{Widget: widget.Text("## API Gateways", 24, 1)},
		{ID: "apigateway.requests", Widget: widget.Metric(apis.Requests(region), 8, 5)},
		{ID: "apigateway.errors4xx", Widget: widget.Metric(apis.Errors4xx(region), 8, 5)},
		{ID: "apigateway.errors5xx", Widget: widget.Metric(apis.Errors5xx(region), 8, 5)},

		{Widget: widget.Text("## Lambdas", 24, 1)},
		{ID: "lambda.invocations", Widget: widget.Metric(lambdas.Invocations(region).LegendRight(), 12, 5)},
		{ID: "lambda.errors", Widget: widget.Metric(lambdas.Errors(region).LegendRight(), 12, 5)},

		{Widget: widget.Text("## S3 Buckets", 24, 1)},
		{ID: "s3.size", Widget: widget.Metric(buckets.Size(region).ShowUnits().LegendRight(), 12, 5)},
		{ID: "s3.objects", Widget: widget.Metric(buckets.Objects(region).ShowUnits().LegendRight(), 12, 5)},
	}
}

func widgetValues(items []dashboardWidget) []widget.Widget {
	widgets := make([]widget.Widget, len(items))
	for i, item := range items {
		widgets[i] = item.Widget
	}
	return widgets
}

type explainSelectors []string

func (selectors *explainSelectors) String() string {
	return strings.Join(*selectors, ",")
}

func (selectors *explainSelectors) Set(value string) error {
	for _, selector := range strings.Split(value, ",") {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			return errors.New("widget id cannot be empty")
		}
		*selectors = append(*selectors, selector)
	}
	return nil
}

func validateExplainSelectors(selectors []string) error {
	valid := make(map[string]bool, len(metricWidgetIDs))
	for _, id := range metricWidgetIDs {
		valid[id] = true
	}
	for _, selector := range selectors {
		if selector != "all" && !valid[selector] {
			return fmt.Errorf("unknown widget %q; valid widgets: %s", selector, strings.Join(metricWidgetIDs, ", "))
		}
	}
	return nil
}

type resourceSelection struct {
	amplify    bool
	apigateway bool
	lambda     bool
	s3         bool
}

func resourcesForSelectors(selectors []string) resourceSelection {
	all := resourceSelection{amplify: true, apigateway: true, lambda: true, s3: true}
	if len(selectors) == 0 {
		return all
	}

	var selected resourceSelection
	for _, selector := range selectors {
		if selector == "all" {
			return all
		}

		family, _, _ := strings.Cut(selector, ".")
		switch family {
		case "amplify":
			selected.amplify = true
		case "apigateway":
			selected.apigateway = true
		case "lambda":
			selected.lambda = true
		case "s3":
			selected.s3 = true
		}
	}
	return selected
}

func selectMetricWidgets(items []dashboardWidget, selectors []string) ([]dashboardWidget, error) {
	available := make(map[string]dashboardWidget)
	for _, item := range items {
		if item.ID != "" {
			available[item.ID] = item
		}
	}

	wanted := make(map[string]bool)
	for _, selector := range selectors {
		if selector == "all" {
			for id := range available {
				wanted[id] = true
			}
			continue
		}
		if _, ok := available[selector]; !ok {
			valid := make([]string, 0, len(available))
			for id := range available {
				valid = append(valid, id)
			}
			sort.Strings(valid)
			return nil, fmt.Errorf("unknown widget %q; valid widgets: %s", selector, strings.Join(valid, ", "))
		}
		wanted[selector] = true
	}

	selected := make([]dashboardWidget, 0, len(wanted))
	for _, item := range items {
		if wanted[item.ID] {
			selected = append(selected, item)
		}
	}
	return selected, nil
}

func explainWidgets(
	ctx context.Context,
	stdout io.Writer,
	cfg aws.Config,
	items []dashboardWidget,
	since time.Duration,
	modelSource string,
	verbose bool,
) error {
	client := cloudwatch.NewFromConfig(cfg)
	end := time.Now().UTC()
	start := end.Add(-since)

	type renderedWidget struct {
		item       dashboardWidget
		properties widget.Properties
		definition []byte
		image      []byte
	}
	rendered := make([]renderedWidget, 0, len(items))

	for _, item := range items {
		properties, ok := item.Widget.Properties.(widget.Properties)
		if !ok {
			return fmt.Errorf("widget %q does not contain metric properties", item.ID)
		}

		renderCtx, cancel := context.WithTimeout(ctx, time.Minute)
		image, err := dashboard.MetricWidgetImage(renderCtx, client, properties, start, end)
		cancel()
		if err != nil {
			return fmt.Errorf("rendering widget %q: %w", item.ID, err)
		}

		definition, err := json.Marshal(properties)
		if err != nil {
			return fmt.Errorf("marshaling widget %q: %w", item.ID, err)
		}
		rendered = append(rendered, renderedWidget{
			item:       item,
			properties: properties,
			definition: definition,
			image:      image,
		})
	}

	log.Printf("loading Kronk model %q", modelSource)
	var progress io.Writer
	if verbose {
		progress = os.Stderr
	}
	explainer, err := explain.NewKronk(ctx, modelSource, progress)
	if err != nil {
		return fmt.Errorf("loading Kronk model: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := explainer.Close(closeCtx); err != nil {
			log.Printf("unloading Kronk model: %v", err)
		}
	}()

	for i, rendered := range rendered {
		requestCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		explanation, err := explainer.Explain(requestCtx, explain.Request{
			WidgetID:   rendered.item.ID,
			Title:      rendered.properties.Title,
			Start:      start,
			End:        end,
			Definition: rendered.definition,
			Image:      rendered.image,
		})
		cancel()
		if err != nil {
			return fmt.Errorf("explaining widget %q: %w", rendered.item.ID, err)
		}

		if i > 0 {
			fmt.Fprintln(stdout)
		}
		fmt.Fprintf(stdout, "## %s — %s\n\n", rendered.item.ID, rendered.properties.Title)
		fmt.Fprintf(stdout, "Time range: %s to %s\n\n", start.Format(time.RFC3339), end.Format(time.RFC3339))
		fmt.Fprintln(stdout, explanation)
	}

	return nil
}

// getResources concurrently retrieves selected AWS resources matching tags. It
// returns the first error encountered, if any.
func getResources(
	ctx context.Context,
	cfg aws.Config,
	tags resource.Tags,
	selected resourceSelection,
) (amplify.Apps, apigateway.RESTs, lambda.Lambdas, s3.Buckets, error) {
	var (
		apps    amplify.Apps
		apis    apigateway.RESTs
		lambdas lambda.Lambdas
		buckets s3.Buckets
	)

	g, gctx := errgroup.WithContext(ctx)
	if selected.amplify {
		g.Go(func() (err error) {
			apps, err = amplify.Get(gctx, cfg, tags)
			log.Printf("found %d amplify apps\n", len(apps))
			return err
		})
	}
	if selected.apigateway {
		g.Go(func() (err error) {
			apis, err = apigateway.Get(gctx, cfg, tags)
			log.Printf("found %d api gateways\n", len(apis))
			return err
		})
	}
	if selected.lambda {
		g.Go(func() (err error) {
			lambdas, err = lambda.Get(gctx, cfg, tags)
			log.Printf("found %d lambdas\n", len(lambdas))
			return err
		})
	}
	if selected.s3 {
		g.Go(func() (err error) {
			buckets, err = s3.Get(gctx, cfg, tags)
			log.Printf("found %d s3 buckets\n", len(buckets))
			return err
		})
	}
	return apps, apis, lambdas, buckets, g.Wait()
}
