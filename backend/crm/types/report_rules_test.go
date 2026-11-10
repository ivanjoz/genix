package types

import (
	"slices"
	"testing"
)

func TestReportablePipelineIDs(t *testing.T) {
	member := CrmPipelineMember{UserID: 7, Level: PipelineLevelView}
	pipelines := []CrmPipeline{
		{ID: 3, Status: CrmStatusActive, Members: []CrmPipelineMember{member}},
		{ID: 1, Status: CrmStatusActive},
		{ID: 2, Status: CrmStatusDeleted, Members: []CrmPipelineMember{member}},
	}
	if got := ReportablePipelineIDs(pipelines, 7, false); !slices.Equal(got, []int16{3}) {
		t.Fatalf("a member reports only on active pipelines they belong to: %v", got)
	}
	if got := ReportablePipelineIDs(pipelines, 7, true); !slices.Equal(got, []int16{1, 2, 3}) {
		t.Fatalf("the full-report access reaches every pipeline: %v", got)
	}
	if got := ReportablePipelineIDs(pipelines, 99, false); len(got) != 0 {
		t.Fatalf("a non-member reports on nothing: %v", got)
	}
}
