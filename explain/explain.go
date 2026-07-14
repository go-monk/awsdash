// Package explain analyzes rendered CloudWatch metric widgets with a local
// multimodal model.
package explain

import (
	"context"
	"time"
)

type Request struct {
	WidgetID   string
	Title      string
	Start      time.Time
	End        time.Time
	Definition []byte
	Image      []byte
}

type Explainer interface {
	Explain(context.Context, Request) (string, error)
	Close(context.Context) error
}
