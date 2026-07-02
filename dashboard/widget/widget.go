package widget

const (
	Period = 60

	// View types
	ViewTimeSeries = "timeSeries"

	// Statistic types
	StatSum     = "Sum"
	StatAverage = "Average"
	statMax     = "Maximum"
	statMin     = "Minimum"
	statP90     = "p90"
	statP75     = "p75"

	// Metric property keys
	KeyColor  = "color"
	KeyLabel  = "label"
	KeyStat   = "stat"
	keyRegion = "region"
)

// Colors provides distinct colors for multi-resource widgets, one color per resource.
var Colors = []string{
	"#1f77b4", // blue
	"#2ca02c", // green
	"#ff7f0e", // orange
	"#9467bd", // purple
	"#8c564b", // brown
	"#e377c2", // pink
	"#17becf", // teal
	"#bcbd22", // yellow-green
	"#7f7f7f", // gray
	"#d62728", // red
}

type Widget struct {
	Type       string `json:"type"`
	Height     int    `json:"height"`
	Width      int    `json:"width"`
	Properties any    `json:"properties"`
}

func Text(markdown string, width, height int) Widget {
	return Widget{
		Type:       "text",
		Width:      width,
		Height:     height,
		Properties: map[string]string{"markdown": markdown},
	}
}

func Metric(properties Properties, width, height int) Widget {
	return Widget{
		Type:       "metric",
		Width:      width,
		Height:     height,
		Properties: properties,
	}
}
