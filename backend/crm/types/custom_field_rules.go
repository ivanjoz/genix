package types

import (
	"fmt"
	"slices"
	"strings"
)

// IsValidCustomFieldType reports whether the type is one of the catalog's ten.
func IsValidCustomFieldType(fieldType int8) bool {
	return fieldType >= CustomFieldText && fieldType <= CustomFieldUser
}

func customFieldHasOptions(fieldType int8) bool {
	return fieldType == CustomFieldSelect || fieldType == CustomFieldMultiSelect
}

// ApplyCustomFieldEdit validates a POST.crm-custom-field payload against the stored record (zero
// for a new field) and returns the record to save. Option IDs are server-allocated and options
// are soft-deleted, never removed, because opportunities still reference them.
// hasStoredValues says whether any opportunity holds a value for the field.
func ApplyCustomFieldEdit(stored, payload CrmCustomField, hasStoredValues bool) (CrmCustomField, error) {
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return CrmCustomField{}, fmt.Errorf("El nombre del campo es obligatorio.")
	}
	if !IsValidCustomFieldType(payload.Type) {
		return CrmCustomField{}, fmt.Errorf("El tipo de campo %d no es válido.", payload.Type)
	}
	if stored.ID != 0 && stored.Type != payload.Type && hasStoredValues {
		return CrmCustomField{}, fmt.Errorf("No se puede cambiar el tipo de un campo que ya tiene valores.")
	}

	savedOptions := []CrmCustomFieldOption{}
	if customFieldHasOptions(payload.Type) {
		storedOptionIDs := map[int16]bool{}
		nextOptionID := int16(1)
		for _, storedOption := range stored.Options {
			storedOptionIDs[storedOption.ID] = true
			nextOptionID = max(nextOptionID, storedOption.ID+1)
		}
		payloadOptionIDs := map[int16]bool{}
		for _, option := range payload.Options {
			option.Name = strings.TrimSpace(option.Name)
			if option.Name == "" {
				return CrmCustomField{}, fmt.Errorf("Todas las opciones necesitan un nombre.")
			}
			if option.ID == 0 {
				option.ID = nextOptionID
				nextOptionID++
			} else if !storedOptionIDs[option.ID] || payloadOptionIDs[option.ID] {
				return CrmCustomField{}, fmt.Errorf("La opción %d no es válida.", option.ID)
			}
			payloadOptionIDs[option.ID] = true
			if option.Status != CrmStatusDeleted {
				option.Status = CrmStatusActive
			}
			savedOptions = append(savedOptions, option)
		}
		for _, storedOption := range stored.Options {
			if !payloadOptionIDs[storedOption.ID] {
				storedOption.Status = CrmStatusDeleted
				savedOptions = append(savedOptions, storedOption)
			}
		}
		activeOptionCount := 0
		for _, option := range savedOptions {
			if option.Status == CrmStatusActive {
				activeOptionCount++
			}
		}
		if activeOptionCount == 0 {
			return CrmCustomField{}, fmt.Errorf("El campo de selección necesita al menos una opción.")
		}
	}

	saved := stored
	saved.Name, saved.Type, saved.Options, saved.Status = name, payload.Type, savedOptions, CrmStatusActive
	// Status 0 deletes an existing field; a new one is always active.
	if stored.ID != 0 && payload.Status == CrmStatusDeleted {
		saved.Status = CrmStatusDeleted
	}
	return saved, nil
}

// ValidateCustomValues checks an opportunity's custom values against the catalog: the field
// exists, the value uses the column of its type, options exist and are active, and every field
// the pipeline requires at the opportunity's stage is present. Values of fields the pipeline does
// not show are allowed (they survive a move between pipelines). activeUserIDs is used for user fields.
func ValidateCustomValues(values []CrmCustomFieldValue, catalog map[int16]CrmCustomField,
	pipeline CrmPipeline, stageID int16, activeUserIDs map[int32]bool) error {

	seenFieldIDs := map[int16]bool{}
	for _, value := range values {
		field, exists := catalog[value.FieldID]
		if !exists || field.Status != CrmStatusActive {
			return fmt.Errorf("El campo personalizado %d no existe.", value.FieldID)
		}
		if seenFieldIDs[value.FieldID] {
			return fmt.Errorf("El campo %q está repetido.", field.Name)
		}
		seenFieldIDs[value.FieldID] = true
		if err := validateCustomValue(value, field, activeUserIDs); err != nil {
			return err
		}
	}
	for _, requiredFieldID := range requiredFieldIDsAtStage(pipeline, stageID) {
		if !seenFieldIDs[requiredFieldID] {
			return fmt.Errorf("El campo %q es obligatorio en esta etapa.", catalog[requiredFieldID].Name)
		}
	}
	return nil
}

// requiredFieldIDsAtStage lists the custom fields an opportunity must have to be in the stage: the
// ones required from that stage's position or an earlier one. A stage that is not active in the
// pipeline requires nothing.
func requiredFieldIDsAtStage(pipeline CrmPipeline, stageID int16) []int16 {
	stagePosition := StagePosition(pipeline, stageID)
	requiredFieldIDs := []int16{}
	for _, pipelineField := range pipeline.Fields {
		requiredFromPosition := StagePosition(pipeline, pipelineField.RequiredFromStageID)
		if stagePosition > 0 && requiredFromPosition > 0 && stagePosition >= requiredFromPosition {
			requiredFieldIDs = append(requiredFieldIDs, pipelineField.FieldID)
		}
	}
	return requiredFieldIDs
}

// requiredStandardFieldsAtStage lists the standard fields (amount, expected close date) required to
// be in the stage: the ones its own list or the list of any earlier stage names.
func requiredStandardFieldsAtStage(pipeline CrmPipeline, stageID int16) []int8 {
	requiredFields := []int8{}
	for _, stage := range pipeline.Stages {
		if stage.Status != CrmStatusActive {
			continue
		}
		for _, standardField := range stage.RequiredStandardFields {
			if !slices.Contains(requiredFields, standardField) {
				requiredFields = append(requiredFields, standardField)
			}
		}
		if stage.ID == stageID {
			return requiredFields
		}
	}
	return []int8{}
}

// MissingRequirements is what an opportunity still lacks to be in a stage.
type MissingRequirements struct {
	// FieldIDs are required custom fields with no value.
	FieldIDs []int16
	// StandardFields are required standard fields (StandardField*) that are empty.
	StandardFields []int8
}

func (missing MissingRequirements) IsEmpty() bool {
	return len(missing.FieldIDs) == 0 && len(missing.StandardFields) == 0
}

// MissingRequiredFields lists what the opportunity lacks to be in the stage of the pipeline, so a
// refused move can tell the UI which fields to ask for. An empty amount or close date is zero.
func MissingRequiredFields(pipeline CrmPipeline, stageID int16, opportunity CrmOpportunity) MissingRequirements {
	present := map[int16]bool{}
	for _, value := range opportunity.CustomValues {
		present[value.FieldID] = true
	}
	missing := MissingRequirements{FieldIDs: []int16{}, StandardFields: []int8{}}
	for _, requiredFieldID := range requiredFieldIDsAtStage(pipeline, stageID) {
		if !present[requiredFieldID] {
			missing.FieldIDs = append(missing.FieldIDs, requiredFieldID)
		}
	}
	for _, standardField := range requiredStandardFieldsAtStage(pipeline, stageID) {
		isEmpty := (standardField == StandardFieldAmount && opportunity.Amount == 0) ||
			(standardField == StandardFieldExpectedCloseDate && opportunity.ExpectedCloseDate == 0)
		if isEmpty {
			missing.StandardFields = append(missing.StandardFields, standardField)
		}
	}
	return missing
}

// checkStageRequirements refuses an opportunity that lacks what the stage requires; the message
// names the missing standard fields and counts the custom ones (the UI knows which they are).
func checkStageRequirements(pipeline CrmPipeline, stageID int16, opportunity CrmOpportunity) error {
	missing := MissingRequiredFields(pipeline, stageID, opportunity)
	if missing.IsEmpty() {
		return nil
	}
	stageName := ""
	if stage, isActive := FindActiveStage(pipeline, stageID); isActive {
		stageName = stage.Name
	}
	missingNames := []string{}
	for _, standardField := range missing.StandardFields {
		if standardField == StandardFieldAmount {
			missingNames = append(missingNames, "monto")
		} else {
			missingNames = append(missingNames, "cierre estimado")
		}
	}
	if len(missing.FieldIDs) > 0 {
		missingNames = append(missingNames, fmt.Sprintf("campos personalizados %v", missing.FieldIDs))
	}
	return fmt.Errorf("Faltan datos obligatorios para la etapa %q: %s.", stageName, strings.Join(missingNames, ", "))
}

func validateCustomValue(value CrmCustomFieldValue, field CrmCustomField, activeUserIDs map[int32]bool) error {
	switch field.Type {
	case CustomFieldText, CustomFieldLongText:
		if strings.TrimSpace(value.Text) == "" {
			return fmt.Errorf("El campo %q no puede estar vacío.", field.Name)
		}
	case CustomFieldCheckbox:
		if value.Number != 0 && value.Number != 1 {
			return fmt.Errorf("El campo %q debe ser 0 o 1.", field.Name)
		}
	case CustomFieldSelect:
		if !customFieldOptionIsActive(field, int16(value.Number)) || value.Number != int64(int16(value.Number)) {
			return fmt.Errorf("La opción elegida en %q no existe.", field.Name)
		}
	case CustomFieldMultiSelect:
		if len(value.OptionIDs) == 0 {
			return fmt.Errorf("El campo %q necesita al menos una opción.", field.Name)
		}
		for _, optionID := range value.OptionIDs {
			if !customFieldOptionIsActive(field, optionID) {
				return fmt.Errorf("Una opción elegida en %q no existe.", field.Name)
			}
		}
	case CustomFieldUser:
		if !activeUserIDs[int32(value.Number)] {
			return fmt.Errorf("El usuario elegido en %q no existe o está inactivo.", field.Name)
		}
	}
	return nil
}

func customFieldOptionIsActive(field CrmCustomField, optionID int16) bool {
	for _, option := range field.Options {
		if option.ID == optionID && option.Status == CrmStatusActive {
			return true
		}
	}
	return false
}
