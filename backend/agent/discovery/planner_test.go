package discovery

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"app/agent/llm"
)

type fakePlannerClient struct {
	requests  []llm.ChatRequest
	responses []string
}

func (fake *fakePlannerClient) Chat(_ context.Context, request llm.ChatRequest) (*llm.ChatResponse, error) {
	index := len(fake.requests)
	fake.requests = append(fake.requests, request)
	return &llm.ChatResponse{Choices: []llm.Choice{{Message: llm.Message{Content: fake.responses[index]}}}}, nil
}

func TestPlannerReturnsValidatedDiscoveryPlan(t *testing.T) {
	encoded, err := json.Marshal(validPlan())
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakePlannerClient{responses: []string{string(encoded)}}
	planner, err := NewPlanner(fake, "muse-spark-1.2-contributor")
	if err != nil {
		t.Fatal(err)
	}
	var observed PlannerAttemptTrace
	plan, err := planner.PlanObserved(context.Background(), validPlannerInput(), func(trace PlannerAttemptTrace) {
		observed = trace
	})
	if err != nil || plan.Goal != GoalViewReport || !plan.Searches.AgentTools.Needed {
		t.Fatalf("unexpected discovery plan: %+v err=%v", plan, err)
	}
	if observed.Attempt != 1 || observed.Response != string(encoded) || observed.Error != "" || len(observed.Messages) != 2 {
		t.Fatalf("planner exchange was not observed: %+v", observed)
	}
	request := fake.requests[0]
	if request.ToolChoice != "" || request.Temperature == nil || *request.Temperature != 0 || request.Reasoning == nil || request.Reasoning.Enabled == nil || *request.Reasoning.Enabled {
		t.Fatalf("planner request is not deterministic: %+v", request)
	}
}

func TestPlannerRetriesInvalidJSONOnce(t *testing.T) {
	encoded, _ := json.Marshal(validPlan())
	fake := &fakePlannerClient{responses: []string{"invalid", string(encoded)}}
	planner, _ := NewPlanner(fake, "muse-spark-1.2-contributor")
	if _, err := planner.Plan(context.Background(), validPlannerInput()); err != nil {
		t.Fatal(err)
	}
	if len(fake.requests) != 2 {
		t.Fatalf("expected one repair attempt, got %d calls", len(fake.requests))
	}
	if feedback := fake.requests[1].Messages[0].Content; !strings.Contains(feedback, "Validation error: decode strict discovery JSON") {
		t.Fatalf("repair attempt did not receive the validation error: %s", feedback)
	}
}

func TestPlannerNormalizesDocumentationQueryBeforeReturn(t *testing.T) {
	planResponse := validPlan()
	planResponse.Goal = GoalManageRecord
	planResponse.Operation = OperationCreate
	planResponse.Domain = "customers"
	planResponse.StandaloneRequest = `Agregar un cliente llamado "Pedro Pascal"`
	planResponse.Entities = []EntityHint{{Type: "customer", Name: "Pedro Pascal"}}
	planResponse.Searches.DocumentationNavigation.Query = planResponse.StandaloneRequest
	planResponse.Searches.AgentTools = SearchDecision{}
	encoded, _ := json.Marshal(planResponse)
	fake := &fakePlannerClient{responses: []string{string(encoded)}}
	planner, _ := NewPlanner(fake, "muse-spark-1.2-contributor")

	plan, err := planner.Plan(context.Background(), validPlannerInput())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Searches.DocumentationNavigation.Query != "Agregar un cliente" {
		t.Fatalf("planner returned instance-specific documentation query: %q", plan.Searches.DocumentationNavigation.Query)
	}
}

func TestPlannerAcceptsConfirmationWithoutDiscovery(t *testing.T) {
	input := validPlannerInput()
	input.CurrentMessage = "si guarda"
	planResponse := validPlan()
	planResponse.Goal = GoalManageRecord
	planResponse.Operation = OperationConfirm
	planResponse.Domain = "inventory"
	planResponse.StandaloneRequest = "Guardar el ajuste de inventario pendiente."
	planResponse.RelatedTurnOffsets = []int{-1}
	planResponse.Searches = SearchPlan{}
	encoded, _ := json.Marshal(planResponse)
	planner, _ := NewPlanner(&fakePlannerClient{responses: []string{string(encoded)}}, "muse-spark-1.2-contributor")

	plan, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Operation != OperationConfirm || plan.Searches.DocumentationNavigation.Needed || plan.Searches.AgentTools.Needed {
		t.Fatalf("confirmation triggered discovery: %+v", plan)
	}
}
