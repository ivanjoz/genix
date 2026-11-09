package types

import (
	"fmt"
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
// the pipeline marks as required is present. Values of fields the pipeline does not show are
// allowed (they survive a move between pipelines). activeUserIDs is used for user fields.
func ValidateCustomValues(values []CrmCustomFieldValue, catalog map[int16]CrmCustomField,
	pipelineFields []CrmPipelineField, activeUserIDs map[int32]bool) error {

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
	for _, pipelineField := range pipelineFields {
		if pipelineField.Required && !seenFieldIDs[pipelineField.FieldID] {
			return fmt.Errorf("El campo %q es obligatorio.", catalog[pipelineField.FieldID].Name)
		}
	}
	return nil
}

// MissingRequiredFields lists the required fields of a target pipeline the values do not cover,
// so a refused move can tell the UI which ones to ask for.
func MissingRequiredFields(values []CrmCustomFieldValue, pipelineFields []CrmPipelineField) []int16 {
	present := map[int16]bool{}
	for _, value := range values {
		present[value.FieldID] = true
	}
	missing := []int16{}
	for _, pipelineField := range pipelineFields {
		if pipelineField.Required && !present[pipelineField.FieldID] {
			missing = append(missing, pipelineField.FieldID)
		}
	}
	return missing
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
