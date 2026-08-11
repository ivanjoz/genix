package routing

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"app/agent/llm"
)

type fakeChatCompleter struct {
	requests  []llm.ChatRequest
	responses []string
	errors    []error
}

func (fake *fakeChatCompleter) Chat(_ context.Context, request llm.ChatRequest) (*llm.ChatResponse, error) {
	index := len(fake.requests)
	fake.requests = append(fake.requests, request)
	if index < len(fake.errors) && fake.errors[index] != nil {
		return nil, fake.errors[index]
	}
	return &llm.ChatResponse{Choices: []llm.Choice{{Message: llm.Message{Content: fake.responses[index]}}}}, nil
}

func verdictJSON(t *testing.T, verdict Verdict) string {
	t.Helper()
	encoded, err := json.Marshal(verdict)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestClassifierUsesFixedMechanicalRequest(t *testing.T) {
	fake := &fakeChatCompleter{responses: []string{verdictJSON(t, validDocumentationVerdict())}}
	classifier, err := NewClassifier(fake, "muse-spark-1.2-contributor")
	if err != nil {
		t.Fatal(err)
	}
	verdict, err := classifier.Classify(context.Background(), validInput())
	if err != nil || verdict.PrimaryIntent != IntentProductKnowledge {
		t.Fatalf("unexpected classifier result: %+v err=%v", verdict, err)
	}
	request := fake.requests[0]
	if request.Model != "muse-spark-1.2-contributor" || request.ToolChoice != "" || request.MaxTokens != classifierMaxTokens {
		t.Fatalf("unexpected classifier request: %+v", request)
	}
	if request.Temperature == nil || *request.Temperature != 0 || request.Reasoning == nil || request.Reasoning.Enabled == nil || *request.Reasoning.Enabled {
		t.Fatalf("classifier must force deterministic minimal reasoning: %+v", request)
	}
}

func TestClassifierRetriesMalformedJSONOnce(t *testing.T) {
	fake := &fakeChatCompleter{responses: []string{"not-json", verdictJSON(t, validDocumentationVerdict())}}
	classifier, _ := NewClassifier(fake, "muse-spark-1.2-contributor")
	if _, err := classifier.Classify(context.Background(), validInput()); err != nil {
		t.Fatal(err)
	}
	if len(fake.requests) != 2 {
		t.Fatalf("expected two attempts, got %d", len(fake.requests))
	}
}

func TestClassifierFailsClosedAfterTwoProviderErrors(t *testing.T) {
	fake := &fakeChatCompleter{errors: []error{errors.New("offline"), errors.New("offline")}}
	classifier, _ := NewClassifier(fake, "muse-spark-1.2-contributor")
	if _, err := classifier.Classify(context.Background(), validInput()); err == nil {
		t.Fatal("classifier must fail closed")
	}
}

func TestDecodeVerdictRejectsUnknownFieldsAndProse(t *testing.T) {
	valid := verdictJSON(t, validDocumentationVerdict())
	if _, err := decodeVerdict(`{"schema":1,"unknown":true}`); err == nil {
		t.Fatal("unknown fields must be rejected")
	}
	if _, err := decodeVerdict("Here is the result: " + valid); err == nil {
		t.Fatal("prose around JSON must be rejected")
	}
}
