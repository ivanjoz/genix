package types

import (
	"fmt"
	"strings"
)

// FindActiveStage returns the stage with that ID when the pipeline still has it active.
func FindActiveStage(pipeline CrmPipeline, stageID int16) (CrmPipelineStage, bool) {
	for _, stage := range pipeline.Stages {
		if stage.ID == stageID && stage.Status == CrmStatusActive {
			return stage, true
		}
	}
	return CrmPipelineStage{}, false
}

// PlanOpportunityCreate validates a new opportunity against its pipeline and returns the record to
// insert. The stage log row is built by the caller once the insert has given the opportunity its ID
// (StageLogOfCreation).
func PlanOpportunityCreate(payload CrmOpportunity, pipeline CrmPipeline, userID, now int32) (CrmOpportunity, error) {
	level := MemberLevel(pipeline, userID)
	if !CanCreateOpportunity(level) {
		return CrmOpportunity{}, fmt.Errorf("No tiene permiso para crear oportunidades en este pipeline.")
	}
	if pipeline.Status != CrmStatusActive {
		return CrmOpportunity{}, fmt.Errorf("El pipeline no existe.")
	}
	_, isActive := FindActiveStage(pipeline, payload.StageID)
	if !isActive {
		return CrmOpportunity{}, fmt.Errorf("La etapa elegida no existe en el pipeline.")
	}
	if err := validateOpportunityFields(&payload); err != nil {
		return CrmOpportunity{}, err
	}
	if err := checkStageRequirements(pipeline, payload.StageID, payload); err != nil {
		return CrmOpportunity{}, err
	}
	if payload.OwnerUserID == 0 {
		payload.OwnerUserID = userID
	}
	if payload.OwnerUserID != userID && !CanChangeOpportunityOwner(level) {
		return CrmOpportunity{}, fmt.Errorf("Solo el nivel Completo puede asignar otro responsable.")
	}
	payload.PipelineID = pipeline.ID
	payload.Outcome, payload.LostReasonID, payload.LostNote, payload.ClosedTime = OpportunityOutcomeOpen, 0, "", 0
	payload.PreviousPipelineID, payload.PreviousStageID = 0, 0
	payload.Created, payload.CreatedBy, payload.StageEnteredTime = now, userID, now
	payload.UpdatedBy = userID
	return payload, nil
}

// StageLogOfCreation is the first log row of an opportunity: from nowhere into its first stage.
func StageLogOfCreation(opportunity CrmOpportunity, pipeline CrmPipeline, userID, now int32) CrmOpportunityStageLog {
	return BuildStageLogRow(StageLogKindCreated, CrmOpportunity{}, CrmPipeline{}, opportunity, pipeline, userID, now)
}

// PlanOpportunityEdit validates an edit of the opportunity's own fields against the stored record
// and returns the record to update. Pipeline and stage are never changed here: moves go through
// PlanOpportunityMove so they are logged.
func PlanOpportunityEdit(stored, payload CrmOpportunity, pipeline CrmPipeline, userID int32) (CrmOpportunity, error) {
	level := MemberLevel(pipeline, userID)
	if !CanEditOpportunity(level, stored, userID) {
		return CrmOpportunity{}, fmt.Errorf("No tiene permiso para editar esta oportunidad.")
	}
	if stored.Outcome != OpportunityOutcomeOpen {
		return CrmOpportunity{}, fmt.Errorf("La oportunidad está cerrada; reábrala para editarla.")
	}
	if err := validateOpportunityFields(&payload); err != nil {
		return CrmOpportunity{}, err
	}
	if payload.OwnerUserID == 0 {
		payload.OwnerUserID = stored.OwnerUserID
	}
	if payload.OwnerUserID != stored.OwnerUserID && !CanChangeOpportunityOwner(level) {
		return CrmOpportunity{}, fmt.Errorf("Solo el nivel Completo puede cambiar el responsable.")
	}
	edited := stored
	edited.Title, edited.ClientID, edited.OwnerUserID = payload.Title, payload.ClientID, payload.OwnerUserID
	edited.Amount, edited.Probability, edited.ExpectedCloseDate = payload.Amount, payload.Probability, payload.ExpectedCloseDate
	edited.CustomValues, edited.UpdatedBy = payload.CustomValues, userID
	// An edit cannot empty a field the opportunity's current stage requires.
	if err := checkStageRequirements(pipeline, stored.StageID, edited); err != nil {
		return CrmOpportunity{}, err
	}
	return edited, nil
}

func validateOpportunityFields(opportunity *CrmOpportunity) error {
	opportunity.Title = strings.TrimSpace(opportunity.Title)
	if opportunity.Title == "" {
		return fmt.Errorf("El título de la oportunidad es obligatorio.")
	}
	if opportunity.ClientID <= 0 {
		return fmt.Errorf("El cliente de la oportunidad es obligatorio.")
	}
	if opportunity.Amount < 0 {
		return fmt.Errorf("El monto no puede ser negativo.")
	}
	if opportunity.Probability < 0 || opportunity.Probability > 100 {
		return fmt.Errorf("La probabilidad debe estar entre 0 y 100.")
	}
	return nil
}

// PlanOpportunityMove applies the §2 rules to a stage or pipeline move and returns the moved
// opportunity with its log row. target is the pipeline the opportunity ends in (the source
// pipeline itself for a stage-only move).
func PlanOpportunityMove(opportunity CrmOpportunity, source, target CrmPipeline, targetStageID int16, userID, now int32) (CrmOpportunity, CrmOpportunityStageLog, error) {
	if opportunity.PendingApprovalTaskID != 0 {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, fmt.Errorf("La oportunidad espera una aprobación; cancele la solicitud o espere la decisión.")
	}
	sourceLevel, targetLevel := MemberLevel(source, userID), MemberLevel(target, userID)
	isCrossPipeline := source.ID != target.ID
	if isCrossPipeline && !CanMoveToPipeline(sourceLevel, targetLevel) {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, fmt.Errorf("Necesita nivel Editar en el pipeline de origen y en el de destino.")
	}
	if !isCrossPipeline && !CanMoveOpportunity(sourceLevel) {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, fmt.Errorf("No tiene permiso para mover esta oportunidad.")
	}
	stage, err := validateMoveTarget(opportunity, target, targetStageID)
	if err != nil {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, err
	}
	moved, stageLog := applyMove(opportunity, source, target, stage, userID, now)
	return moved, stageLog, nil
}

// validateMoveTarget holds the state checks a move into the stage must pass, however it came about
// (a direct move or an approved one): the opportunity is open, the move changes something, the
// target pipeline and stage are active, and the stage's required data is complete. It returns the stage.
func validateMoveTarget(opportunity CrmOpportunity, target CrmPipeline, targetStageID int16) (CrmPipelineStage, error) {
	if opportunity.Outcome != OpportunityOutcomeOpen {
		return CrmPipelineStage{}, fmt.Errorf("La oportunidad está cerrada; reábrala para moverla.")
	}
	if opportunity.PipelineID == target.ID && opportunity.StageID == targetStageID {
		return CrmPipelineStage{}, fmt.Errorf("La oportunidad ya está en esa etapa.")
	}
	if target.Status != CrmStatusActive {
		return CrmPipelineStage{}, fmt.Errorf("El pipeline de destino no existe.")
	}
	stage, isActive := FindActiveStage(target, targetStageID)
	if !isActive {
		return CrmPipelineStage{}, fmt.Errorf("La etapa de destino no existe o está desactivada.")
	}
	// Entering the stage needs everything required from it or from an earlier stage of the target.
	if err := checkStageRequirements(target, targetStageID, opportunity); err != nil {
		return CrmPipelineStage{}, err
	}
	return stage, nil
}

// applyMove builds the moved record and its log row; the checks were made by the caller.
func applyMove(opportunity CrmOpportunity, source, target CrmPipeline, stage CrmPipelineStage, userID, now int32) (CrmOpportunity, CrmOpportunityStageLog) {
	moved := opportunity
	moved.PreviousPipelineID, moved.PreviousStageID = opportunity.PipelineID, opportunity.StageID
	moved.PipelineID, moved.StageID = target.ID, stage.ID
	moved.StageEnteredTime, moved.Probability, moved.UpdatedBy = now, stage.Probability, userID
	moved.PendingApprovalTaskID = 0
	return moved, BuildStageLogRow(StageLogKindMoved, opportunity, source, moved, target, userID, now)
}

// PlanOpportunityClose closes the opportunity as won or lost, or reopens it (outcome 0). A lost
// outcome needs an active reason; reopening needs the stage to still exist.
func PlanOpportunityClose(opportunity CrmOpportunity, pipeline CrmPipeline, payload CrmOpportunityClosePayload,
	lostReasonIsActive bool, userID, now int32) (CrmOpportunity, CrmOpportunityStageLog, error) {

	if !CanMoveOpportunity(MemberLevel(pipeline, userID)) {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, fmt.Errorf("No tiene permiso para cerrar o reabrir esta oportunidad.")
	}
	if opportunity.PendingApprovalTaskID != 0 {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, fmt.Errorf("La oportunidad espera una aprobación; cancele la solicitud o espere la decisión.")
	}
	if payload.Outcome < OpportunityOutcomeOpen || payload.Outcome > OpportunityOutcomeLost {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, fmt.Errorf("El resultado %d no es válido.", payload.Outcome)
	}
	if payload.Outcome == opportunity.Outcome {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, fmt.Errorf("La oportunidad ya tiene ese resultado.")
	}
	if payload.Outcome == OpportunityOutcomeLost && !lostReasonIsActive {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, fmt.Errorf("Elija un motivo de pérdida válido.")
	}
	if payload.Outcome == OpportunityOutcomeOpen {
		if _, isActive := FindActiveStage(pipeline, opportunity.StageID); !isActive {
			return CrmOpportunity{}, CrmOpportunityStageLog{}, fmt.Errorf("La etapa de la oportunidad ya no existe; no se puede reabrir.")
		}
	}

	closed := opportunity
	closed.Outcome, closed.UpdatedBy = payload.Outcome, userID
	closed.LostReasonID, closed.LostNote, closed.ClosedTime = 0, "", 0
	if payload.Outcome != OpportunityOutcomeOpen {
		closed.ClosedTime = now
	}
	if payload.Outcome == OpportunityOutcomeLost {
		closed.LostReasonID, closed.LostNote = payload.LostReasonID, strings.TrimSpace(payload.LostNote)
	}
	if payload.Outcome == OpportunityOutcomeOpen {
		// A reopened card re-enters its stage now, so the stale flag starts again.
		closed.StageEnteredTime = now
	}
	closed.SelfParse()
	return closed, BuildStageLogRow(StageLogKindClosed, opportunity, pipeline, closed, pipeline, userID, now), nil
}
