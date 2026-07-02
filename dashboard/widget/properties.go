package widget

type Properties struct {
	View    string  `json:"view"`
	Stacked bool    `json:"stacked"`
	Region  string  `json:"region"`
	Period  int     `json:"period"`
	Title   string  `json:"title"`
	Metrics [][]any `json:"metrics"`
	YAxis   *YAxis  `json:"yAxis,omitempty"`
	Stat    string  `json:"stat,omitempty"`
	Legend  Legend  `json:"legend,omitempty"`
}

type YAxis struct {
	Left *YAxisSide `json:"left,omitempty"`
}

type YAxisSide struct {
	Label     string `json:"label,omitempty"`
	ShowUnits bool   `json:"showUnits"`
}

type Legend struct {
	Position string `json:"position"`
}

func (p Properties) ShowUnits() Properties {
	p.YAxis.Left.ShowUnits = true
	return p
}

func (p Properties) LegendRight() Properties {
	p.Legend.Position = "right"
	return p
}

func NewProperties(title, region string, metrics [][]any) Properties {
	return Properties{
		View:    ViewTimeSeries,
		Stacked: false,
		Period:  Period,
		Title:   title,
		Region:  region,
		Metrics: metrics,
		YAxis: &YAxis{
			Left: &YAxisSide{
				ShowUnits: false,
			},
		},
	}
}
