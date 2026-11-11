package types

import (
	"fmt"
	"strings"
)

// MemberLevel returns the user's level in the pipeline, 0 when the user is not a member.
func MemberLevel(pipeline CrmPipeline, userID int32) int8 {
	for _, member := range pipeline.Members {
		if member.UserID == userID {
			return member.Level
		}
	}
	return 0
}

// CanCreateOpportunity: Edit and Full members can create.
func CanCreateOpportunity(level int8) bool {
	return level >= PipelineLevelEdit
}

// CanMoveOpportunity covers moving between stages, and closing or reopening: any Edit or Full
// member can, whoever created or owns the opportunity. The same rule applies to the source
// pipeline of a cross-pipeline move; the target pipeline is checked with CanCreateOpportunity.
func CanMoveOpportunity(level int8) bool {
	return level >= PipelineLevelEdit
}

// CanEditOpportunity covers the opportunity's own fields: Full edits any, Edit only the ones the
// user created or owns.
func CanEditOpportunity(level int8, opportunity CrmOpportunity, userID int32) bool {
	if level >= PipelineLevelFull {
		return true
	}
	return level == PipelineLevelEdit && (opportunity.CreatedBy == userID || opportunity.OwnerUserID == userID)
}

// CanChangeOpportunityOwner: reassigning a lead is Full level only.
func CanChangeOpportunityOwner(level int8) bool {
	return level >= PipelineLevelFull
}

// CanMoveToPipeline is the rule for a cross-pipeline move: it needs the move right on the source
// and the right to create on the target, or a user could push an opportunity into a pipeline
// they cannot work in.
func CanMoveToPipeline(sourceLevel, targetLevel int8) bool {
	return CanMoveOpportunity(sourceLevel) && CanCreateOpportunity(targetLevel)
}

// ApplyPipelineEdit validates a POST.crm-pipeline payload against the stored record and returns
// the record to save. Members always come from the stored record, never from the payload.
// stageHoldsOpenOpportunities answers whether an existing stage still has open opportunities;
// knownFieldIDs are the catalog's active custom field IDs. stored is a zero value for a new pipeline.
func ApplyPipelineEdit(stored, payload CrmPipeline, stageHoldsOpenOpportunities func(stageID int16) bool,
	knownFieldIDs map[int16]bool) (CrmPipeline, error) {

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return CrmPipeline{}, fmt.Errorf("El nombre del pipeline es obligatorio.")
	}

	storedStageIDs := map[int16]bool{}
	nextStageID := int16(1)
	for _, storedStage := range stored.Stages {
		storedStageIDs[storedStage.ID] = true
		nextStageID = max(nextStageID, storedStage.ID+1)
	}

	// Stage IDs belong to the server: an ID the client sends must already exist.
	savedStages := make([]CrmPipelineStage, 0, len(payload.Stages))
	payloadStageIDs := map[int16]bool{}
	activeStageCount := 0
	for _, stage := range payload.Stages {
		stage.Name = strings.TrimSpace(stage.Name)
		if stage.Name == "" {
			return CrmPipeline{}, fmt.Errorf("Todas las etapas necesitan un nombre.")
		}
		if stage.Probability < 0 || stage.Probability > 100 {
			return CrmPipeline{}, fmt.Errorf("La probabilidad de la etapa %q debe estar entre 0 y 100.", stage.Name)
		}
		if stage.StaleDays < 0 {
			return CrmPipeline{}, fmt.Errorf("Los días para marcar estancada la etapa %q no pueden ser negativos.", stage.Name)
		}
		if stage.ForecastCategory == 0 {
			stage.ForecastCategory = ForecastCategoryPipeline
		}
		if stage.ForecastCategory < ForecastCategoryPipeline || stage.ForecastCategory > ForecastCategoryCommit {
			return CrmPipeline{}, fmt.Errorf("La categoría de forecast de la etapa %q no es válida (1 pipeline, 2 mejor caso, 3 compromiso).", stage.Name)
		}
		standardFieldsSeen := map[int8]bool{}
		for _, standardField := range stage.RequiredStandardFields {
			if standardField != StandardFieldAmount && standardField != StandardFieldExpectedCloseDate {
				return CrmPipeline{}, fmt.Errorf("El campo estándar %d de la etapa %q no es válido (1 monto, 2 cierre estimado).", standardField, stage.Name)
			}
			if standardFieldsSeen[standardField] {
				return CrmPipeline{}, fmt.Errorf("El campo estándar %d está repetido en la etapa %q.", standardField, stage.Name)
			}
			standardFieldsSeen[standardField] = true
		}
		if stage.ID == 0 {
			stage.ID = nextStageID
			nextStageID++
		} else if !storedStageIDs[stage.ID] {
			return CrmPipeline{}, fmt.Errorf("La etapa %d no existe en el pipeline.", stage.ID)
		}
		if payloadStageIDs[stage.ID] {
			return CrmPipeline{}, fmt.Errorf("La etapa %d está repetida.", stage.ID)
		}
		payloadStageIDs[stage.ID] = true
		taskTemplates, err := normalizeTaskTemplates(stage.TaskTemplates, storedTemplatesOfStage(stored, stage.ID), stage.Name)
		if err != nil {
			return CrmPipeline{}, err
		}
		stage.TaskTemplates = taskTemplates
		if stage.Status != CrmStatusDeleted {
			stage.Status = CrmStatusActive
			activeStageCount++
		}
		savedStages = append(savedStages, stage)
	}
	if activeStageCount == 0 {
		return CrmPipeline{}, fmt.Errorf("El pipeline necesita al menos una etapa activa.")
	}

	// A stage that disappears or is soft-deleted must be empty (D5). Omitting a stored stage
	// would lose its label, so it is kept as deleted.
	savedStatusByID := map[int16]int8{}
	for _, savedStage := range savedStages {
		savedStatusByID[savedStage.ID] = savedStage.Status
	}
	for _, storedStage := range stored.Stages {
		isInPayload := payloadStageIDs[storedStage.ID]
		isStillActive := isInPayload && savedStatusByID[storedStage.ID] == CrmStatusActive
		if storedStage.Status == CrmStatusActive && !isStillActive && stageHoldsOpenOpportunities(storedStage.ID) {
			return CrmPipeline{}, fmt.Errorf("La etapa %q todavía tiene oportunidades abiertas; muévalas antes de eliminarla.", storedStage.Name)
		}
		if !isInPayload {
			storedStage.Status = CrmStatusDeleted
			savedStages = append(savedStages, storedStage)
		}
	}

	seenFieldIDs := map[int16]bool{}
	for _, field := range payload.Fields {
		if !knownFieldIDs[field.FieldID] {
			return CrmPipeline{}, fmt.Errorf("El campo personalizado %d no existe.", field.FieldID)
		}
		if seenFieldIDs[field.FieldID] {
			return CrmPipeline{}, fmt.Errorf("El campo personalizado %d está repetido.", field.FieldID)
		}
		seenFieldIDs[field.FieldID] = true
		// The stage a field is required from must stay active: a soft-deleted one would silently
		// drop the rule. A new pipeline's stages get IDs 1..n in order, so those may be named too.
		if field.RequiredFromStageID != 0 && savedStatusByID[field.RequiredFromStageID] != CrmStatusActive {
			return CrmPipeline{}, fmt.Errorf("El campo personalizado %d es obligatorio desde una etapa que no existe o fue eliminada.", field.FieldID)
		}
	}

	goalPeriodType := payload.GoalPeriodType
	if goalPeriodType == 0 {
		goalPeriodType = GoalPeriodMonthly
	}
	if goalPeriodType < GoalPeriodMonthly || goalPeriodType > GoalPeriodDailyByWeek {
		return CrmPipeline{}, fmt.Errorf("El periodo de metas %d no es válido (1 mensual, 2 semanal, 3 diaria por mes, 4 diaria por semana).", goalPeriodType)
	}

	saved := stored
	saved.GoalPeriodType = goalPeriodType
	saved.Name, saved.Description, saved.Color = name, strings.TrimSpace(payload.Description), payload.Color
	saved.SortOrder, saved.Stages, saved.Fields = payload.SortOrder, savedStages, payload.Fields
	saved.Status = CrmStatusActive
	return saved, nil
}

// storedTemplatesOfStage returns the stored task templates of a stage (none for a new one).
func storedTemplatesOfStage(stored CrmPipeline, stageID int16) []CrmStageTaskTemplate {
	for _, storedStage := range stored.Stages {
		if storedStage.ID == stageID {
			return storedStage.TaskTemplates
		}
	}
	return nil
}

// ValidatePipelineMembers checks a POST.crm-pipeline-members list: known active users, levels 1–3,
// no duplicates.
func ValidatePipelineMembers(members []CrmPipelineMember, activeUserIDs map[int32]bool) error {
	seenUserIDs := map[int32]bool{}
	for _, member := range members {
		if member.Level < PipelineLevelView || member.Level > PipelineLevelFull {
			return fmt.Errorf("El nivel %d del usuario %d no es válido (1 ver, 2 editar, 3 completo).", member.Level, member.UserID)
		}
		if !activeUserIDs[member.UserID] {
			return fmt.Errorf("El usuario %d no existe o está inactivo.", member.UserID)
		}
		if seenUserIDs[member.UserID] {
			return fmt.Errorf("El usuario %d está repetido.", member.UserID)
		}
		seenUserIDs[member.UserID] = true
	}
	return nil
}
