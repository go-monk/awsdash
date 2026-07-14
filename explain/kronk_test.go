package explain

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ardanlabs/kronk/sdk/kronk/model"
)

type fakeChatModel struct {
	request  model.D
	response model.ChatResponse
	err      error
}

func (fake *fakeChatModel) Chat(_ context.Context, request model.D) (model.ChatResponse, error) {
	fake.request = request
	return fake.response, fake.err
}

func (*fakeChatModel) Unload(context.Context) error { return nil }

func TestPromptDefinesEvidenceBoundaries(t *testing.T) {
	request := Request{
		WidgetID:   "lambda.errors",
		Title:      "Errors",
		Start:      time.Date(2026, time.July, 14, 6, 0, 0, 0, time.UTC),
		End:        time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC),
		Definition: []byte(`{"region":"eu-central-1"}`),
	}

	got := prompt(request)
	for _, want := range []string{
		"Widget id: lambda.errors",
		"2026-07-14T06:00:00Z to 2026-07-14T12:00:00Z",
		`{"region":"eu-central-1"}`,
		"Do not invent values, events, or root causes",
		"Next checks",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt does not contain %q", want)
		}
	}
}

func TestKronkExplainDisablesThinking(t *testing.T) {
	finishReason := model.FinishReasonStop
	fake := &fakeChatModel{response: model.ChatResponse{
		Choices: []model.Choice{{
			Message:         &model.ResponseMessage{Content: "  No 4xx errors are visible.  "},
			FinishReasonPtr: &finishReason,
		}},
	}}
	explainer := &kronkExplainer{model: fake}

	got, err := explainer.Explain(context.Background(), Request{Image: []byte("png")})
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if got != "No 4xx errors are visible." {
		t.Fatalf("explanation = %q", got)
	}
	if got := fake.request["enable_thinking"]; got != false {
		t.Fatalf("enable_thinking = %#v, want false", got)
	}
	if got := fake.request["max_tokens"]; got != 1024 {
		t.Fatalf("max_tokens = %#v, want 1024", got)
	}
}

func TestKronkExplainReportsReasoningOnlyResponse(t *testing.T) {
	finishReason := model.FinishReasonStop
	fake := &fakeChatModel{response: model.ChatResponse{
		Choices: []model.Choice{{
			Message:         &model.ResponseMessage{Reasoning: "analysis without an answer"},
			FinishReasonPtr: &finishReason,
		}},
	}}
	explainer := &kronkExplainer{model: fake}

	_, err := explainer.Explain(context.Background(), Request{Image: []byte("png")})
	if err == nil {
		t.Fatal("Explain returned nil error for a reasoning-only response")
	}
	if !strings.Contains(err.Error(), "reasoning but no final explanation") {
		t.Fatalf("error = %q", err)
	}
}
