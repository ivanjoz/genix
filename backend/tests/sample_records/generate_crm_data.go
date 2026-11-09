package sample_records

import (
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"strings"
	"time"

	"app/core"
	coreTypes "app/core/types"
	crmModule "app/crm"
	crmTypes "app/crm/types"
	"app/db"
)

// generate_crm_data fills the CRM of the demo company through the production handlers of app/crm,
// with the clock frozen on each simulated moment: custom fields, lost reasons, two pipelines with
// their members, a few prospects, and a few dozen opportunities that are created, moved stage by
// stage, won, lost or reopened over the last months. The stage log therefore comes out the way the
// board's history reads it. A company that already has pipelines is left untouched.

const (
	crmDataOpportunitiesCount       = 36
	crmDataHistoryDays              = 120
	crmDataActorID            int32 = 1
)

var crmDataProspectNames = []string{
	"Frutos del Sur SAC", "Distribuidora Andina EIRL", "Minimarket La Esquina", "Restaurante Sabor Criollo",
	"Hotel Casa Real", "Bodega Don Lucho", "Panadería El Trigal", "Clínica San Judas",
}

var crmDataOpportunityTitles = []string{
	"Pedido mensual de abarrotes", "Abastecimiento de frutas", "Contrato anual de lácteos", "Compra de bebidas para evento",
	"Reposición de limpieza", "Cotización de congelados", "Convenio de entrega semanal", "Pedido de temporada",
	"Equipamiento de cocina", "Suministro para comedor",
}

var crmDataLostReasonNames = []string{"Precio", "Eligió a la competencia", "Sin respuesta", "Sin presupuesto"}

func crmDataCall(actorID int32, route string, handler func(*core.HandlerArgs) core.HandlerResponse, payload any, target any) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body := string(bodyBytes)
	method, _, _ := strings.Cut(route, ".")
	request := core.HandlerArgs{
		Body: &body, Query: map[string]string{}, Route: route, Method: method,
		User: &core.UsuarioToken{CompanyID: sampleCompanyID, ID: actorID},
	}
	response := handler(&request)
	if response.StatusCode != 200 {
		return fmt.Errorf("%s (usuario %d) respondió %d: %s", route, actorID, response.StatusCode, response.Error)
	}
	if target == nil || response.Body == nil {
		return nil
	}
	return json.Unmarshal(*response.Body, target)
}

// GenerateCrmData is the entry point behind `fn-generate-crm-data`.
func GenerateCrmData(args *core.ExecArgs) core.FuncResponse {
	existingPipelines := []crmTypes.CrmPipeline{}
	pipelineQuery := db.Query(&existingPipelines)
	pipelineQuery.CompanyID.Equals(sampleCompanyID)
	if err := pipelineQuery.Exec(); err != nil {
		return args.MakeErr("No se pudieron leer los pipelines:", err)
	}
	for _, pipeline := range existingPipelines {
		if pipeline.Status != crmTypes.CrmStatusActive {
			continue
		}
		return core.FuncResponse{Message: "La empresa ya tiene pipelines de CRM: no se escribió nada."}
	}

	defer core.SetHistoricalUnix(0)
	realNow := time.Now()
	random := mrand.New(mrand.NewSource(sampleDataSeed))
	if err := runCrmData(random, realNow); err != nil {
		return args.MakeErr("Falló la generación del CRM:", err)
	}
	return core.FuncResponse{Message: fmt.Sprintf("CRM generado: 2 pipelines y %d oportunidades.", crmDataOpportunitiesCount)}
}

func runCrmData(random *mrand.Rand, realNow time.Time) error {
	startOfHistory := realNow.AddDate(0, 0, -crmDataHistoryDays)
	core.SetHistoricalUnix(startOfHistory.Unix())

	// Catalogs first: fields and reasons exist before the pipelines that use them.
	fieldNames := []struct {
		name   string
		kind   int8
		option []string
	}{
		{"Origen del contacto", crmTypes.CustomFieldSelect, []string{"Web", "Feria", "Referido", "Llamada en frío"}},
		{"Presupuesto confirmado", crmTypes.CustomFieldCheckbox, nil},
		{"Nota comercial", crmTypes.CustomFieldText, nil},
	}
	customFields := []crmTypes.CrmCustomField{}
	for _, fieldSeed := range fieldNames {
		options := []crmTypes.CrmCustomFieldOption{}
		for _, optionName := range fieldSeed.option {
			options = append(options, crmTypes.CrmCustomFieldOption{Name: optionName, Status: crmTypes.CrmStatusActive})
		}
		saved := crmTypes.CrmCustomField{}
		payload := crmTypes.CrmCustomField{Name: fieldSeed.name, Type: fieldSeed.kind, Options: options, Status: crmTypes.CrmStatusActive}
		if err := crmDataCall(crmDataActorID, "POST.crm-custom-field", crmModule.PostCrmCustomField, payload, &saved); err != nil {
			return err
		}
		customFields = append(customFields, saved)
	}
	lostReasons := []crmTypes.CrmLostReason{}
	for _, reasonName := range crmDataLostReasonNames {
		saved := crmTypes.CrmLostReason{}
		if err := crmDataCall(crmDataActorID, "POST.crm-lost-reason", crmModule.PostCrmLostReason, crmTypes.CrmLostReason{Name: reasonName}, &saved); err != nil {
			return err
		}
		lostReasons = append(lostReasons, saved)
	}

	// Pipelines: the wholesale one asks for the contact origin; the projects one does not.
	stage := func(name, color string, probability int8, staleDays int16) crmTypes.CrmPipelineStage {
		return crmTypes.CrmPipelineStage{Name: name, Color: color, Probability: probability, StaleDays: staleDays, Status: crmTypes.CrmStatusActive}
	}
	wholesale := crmTypes.CrmPipeline{Name: "Ventas mayoristas", Color: "#3b82f6", SortOrder: 1, Status: crmTypes.CrmStatusActive,
		Stages: []crmTypes.CrmPipelineStage{
			stage("Contacto", "#6b7280", 10, 10), stage("Cotización", "#0ea5e9", 30, 14), stage("Negociación", "#f59e0b", 60, 14),
			stage("Pedido de prueba", "#8b5cf6", 80, 21),
		},
		Fields: []crmTypes.CrmPipelineField{{FieldID: customFields[0].ID, Required: true}, {FieldID: customFields[2].ID}}}
	projects := crmTypes.CrmPipeline{Name: "Proyectos especiales", Color: "#10b981", SortOrder: 2, Status: crmTypes.CrmStatusActive,
		Stages: []crmTypes.CrmPipelineStage{
			stage("Levantamiento", "#6b7280", 15, 14), stage("Propuesta", "#0ea5e9", 40, 21), stage("Aprobación", "#f59e0b", 75, 21),
		},
		Fields: []crmTypes.CrmPipelineField{{FieldID: customFields[1].ID}, {FieldID: customFields[2].ID}}}
	pipelines := []crmTypes.CrmPipeline{wholesale, projects}
	for pipelineIndex := range pipelines {
		if err := crmDataCall(crmDataActorID, "POST.crm-pipeline", crmModule.PostCrmPipeline, pipelines[pipelineIndex], &pipelines[pipelineIndex]); err != nil {
			return err
		}
	}

	// Members: the actor is Full (added when the pipeline is created); the next users get Edit and View.
	users := []coreTypes.User{}
	userQuery := db.Query(&users)
	userQuery.Select(userQuery.ID, userQuery.Status).CompanyID.Equals(sampleCompanyID)
	if err := userQuery.Exec(); err != nil {
		return err
	}
	editorIDs := []int32{crmDataActorID}
	members := []crmTypes.CrmPipelineMember{{UserID: crmDataActorID, Level: crmTypes.PipelineLevelFull}}
	for _, user := range users {
		if user.ID == crmDataActorID || user.Status != crmTypes.CrmStatusActive || len(members) >= 4 {
			continue
		}
		level := crmTypes.PipelineLevelEdit
		if len(members) == 3 {
			level = crmTypes.PipelineLevelView
		} else {
			editorIDs = append(editorIDs, user.ID)
		}
		members = append(members, crmTypes.CrmPipelineMember{UserID: user.ID, Level: level})
	}
	for pipelineIndex := range pipelines {
		membersPayload := crmTypes.CrmPipelineMembersPayload{PipelineID: pipelines[pipelineIndex].ID, Members: members}
		if err := crmDataCall(crmDataActorID, "POST.crm-pipeline-members", crmModule.PostCrmPipelineMembers, membersPayload, &pipelines[pipelineIndex]); err != nil {
			return err
		}
	}

	// Clients: the company's own, plus a few prospects that have not bought yet.
	clientIDs, err := ensureCrmDataClients(random)
	if err != nil {
		return err
	}

	for opportunityIndex := 0; opportunityIndex < crmDataOpportunitiesCount; opportunityIndex++ {
		pipeline := pipelines[random.Intn(len(pipelines))]
		if err := runCrmDataOpportunity(random, realNow, pipeline, pipelines, customFields, lostReasons, clientIDs, editorIDs, opportunityIndex); err != nil {
			return fmt.Errorf("oportunidad %d: %w", opportunityIndex, err)
		}
	}
	return nil
}

// ensureCrmDataClients returns the IDs of the company's clients, creating the prospects of the demo.
func ensureCrmDataClients(random *mrand.Rand) ([]int32, error) {
	prospects := []crmTypes.ClientProvider{}
	for _, prospectName := range crmDataProspectNames {
		prospects = append(prospects, crmTypes.ClientProvider{
			Type: crmTypes.ClientProviderTypeClient, Name: prospectName, PersonType: crmTypes.PersonTypeCompany, IsProspect: true,
		})
	}
	saved := []crmTypes.ClientProvider{}
	if err := crmDataCall(crmDataActorID, "POST.client-provider", crmModule.PostClientProviders, prospects, &saved); err != nil {
		return nil, err
	}
	clientIDs := []int32{}
	for _, client := range saved {
		clientIDs = append(clientIDs, client.ID)
	}
	// Existing clients join the pool too, so cards mix leads and customers.
	existing := []crmTypes.ClientProvider{}
	clientQuery := db.Query(&existing)
	clientQuery.Select(clientQuery.ID, clientQuery.Type, clientQuery.Status).CompanyID.Equals(sampleCompanyID).Type.Equals(crmTypes.ClientProviderTypeClient).Limit(10)
	if err := clientQuery.Exec(); err != nil {
		return nil, err
	}
	for _, client := range existing {
		if client.Status == crmTypes.CrmStatusActive {
			clientIDs = append(clientIDs, client.ID)
		}
	}
	random.Shuffle(len(clientIDs), func(left, right int) { clientIDs[left], clientIDs[right] = clientIDs[right], clientIDs[left] })
	return clientIDs, nil
}

// runCrmDataOpportunity lives one opportunity: created on a past day by an Edit/Full member, then
// moved a few stages forward a few days apart, and finally left open, won, lost, or lost and reopened.
func runCrmDataOpportunity(random *mrand.Rand, realNow time.Time, pipeline crmTypes.CrmPipeline, pipelines []crmTypes.CrmPipeline,
	customFields []crmTypes.CrmCustomField, lostReasons []crmTypes.CrmLostReason, clientIDs, editorIDs []int32, opportunityIndex int) error {

	moment := realNow.AddDate(0, 0, -(5 + random.Intn(crmDataHistoryDays-5)))
	freezeAt := func() { core.SetHistoricalUnix(moment.Unix()) }
	advance := func() bool {
		moment = moment.Add(time.Duration(2+random.Intn(8)) * 24 * time.Hour)
		return moment.Before(realNow)
	}

	values := []crmTypes.CrmCustomFieldValue{}
	for _, pipelineField := range pipeline.Fields {
		switch pipelineField.FieldID {
		case customFields[0].ID:
			values = append(values, crmTypes.CrmCustomFieldValue{FieldID: pipelineField.FieldID, Number: int64(1 + random.Intn(4))})
		case customFields[1].ID:
			values = append(values, crmTypes.CrmCustomFieldValue{FieldID: pipelineField.FieldID, Number: int64(random.Intn(2))})
		case customFields[2].ID:
			if random.Intn(2) == 0 {
				values = append(values, crmTypes.CrmCustomFieldValue{FieldID: pipelineField.FieldID, Text: "Seguimiento por WhatsApp"})
			}
		}
	}
	activeStages := pipeline.Stages
	creatorID := editorIDs[random.Intn(len(editorIDs))]
	created := crmTypes.CrmOpportunity{}
	freezeAt()
	err := crmDataCall(creatorID, "POST.crm-opportunity", crmModule.PostCrmOpportunity, crmTypes.CrmOpportunity{
		PipelineID: pipeline.ID, StageID: activeStages[0].ID, ClientID: clientIDs[opportunityIndex%len(clientIDs)],
		Title:  fmt.Sprintf("%s #%d", crmDataOpportunityTitles[random.Intn(len(crmDataOpportunityTitles))], opportunityIndex+1),
		Amount: int32(500_00 + random.Intn(14_500_00)), Probability: activeStages[0].Probability,
		ExpectedCloseDate: core.TimeToFechaUnix(realNow) + int16(10+random.Intn(50)), CustomValues: values,
	}, &created)
	if err != nil {
		return err
	}

	// Walk forward through the stages; each step happens a few days after the last, never in the future.
	reachedLastStage := true
	for stageIndex := 1; stageIndex < len(activeStages); stageIndex++ {
		if random.Intn(100) < 35 || !advance() {
			reachedLastStage = false
			break
		}
		freezeAt()
		if err := crmDataCall(crmDataActorID, "POST.crm-opportunity-move", crmModule.PostCrmOpportunityMove,
			crmTypes.CrmOpportunityMovePayload{ID: created.ID, PipelineID: pipeline.ID, StageID: activeStages[stageIndex].ID}, &created); err != nil {
			return err
		}
	}

	// One in ten wholesale cards crosses to the projects pipeline (the target asks for no required field).
	if pipeline.ID == pipelines[0].ID && random.Intn(10) == 0 && advance() {
		freezeAt()
		if err := crmDataCall(crmDataActorID, "POST.crm-opportunity-move", crmModule.PostCrmOpportunityMove,
			crmTypes.CrmOpportunityMovePayload{ID: created.ID, PipelineID: pipelines[1].ID, StageID: pipelines[1].Stages[0].ID}, &created); err != nil {
			return err
		}
		return nil
	}

	// Outcome: cards at the last stage close more often; the rest mostly stay open on the board.
	closeChance := 20
	if reachedLastStage {
		closeChance = 75
	}
	if random.Intn(100) >= closeChance || !advance() {
		return nil
	}
	freezeAt()
	if random.Intn(100) < 55 {
		return crmDataCall(crmDataActorID, "POST.crm-opportunity-close", crmModule.PostCrmOpportunityClose,
			crmTypes.CrmOpportunityClosePayload{ID: created.ID, Outcome: crmTypes.OpportunityOutcomeWon}, nil)
	}
	lostReason := lostReasons[random.Intn(len(lostReasons))]
	if err := crmDataCall(crmDataActorID, "POST.crm-opportunity-close", crmModule.PostCrmOpportunityClose,
		crmTypes.CrmOpportunityClosePayload{ID: created.ID, Outcome: crmTypes.OpportunityOutcomeLost, LostReasonID: lostReason.ID, LostNote: "Registrado en la simulación"}, nil); err != nil {
		return err
	}
	// A few lost deals come back.
	if random.Intn(100) < 15 && advance() {
		freezeAt()
		return crmDataCall(crmDataActorID, "POST.crm-opportunity-close", crmModule.PostCrmOpportunityClose,
			crmTypes.CrmOpportunityClosePayload{ID: created.ID, Outcome: crmTypes.OpportunityOutcomeOpen}, nil)
	}
	return nil
}
