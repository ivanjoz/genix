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
	crmDataOpportunitiesCount = 300
	crmDataHistoryDays        = 365
	// An opportunity lives 2..30 calendar days from creation to its last change.
	crmDataMaxSpanDays       = 30
	crmDataActorID     int32 = 1
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
		// The forecast category follows how likely the stage is to close.
		category := crmTypes.ForecastCategoryPipeline
		if probability >= 60 {
			category = crmTypes.ForecastCategoryCommit
		} else if probability >= 30 {
			category = crmTypes.ForecastCategoryBestCase
		}
		return crmTypes.CrmPipelineStage{Name: name, Color: color, Probability: probability, StaleDays: staleDays,
			Status: crmTypes.CrmStatusActive, ForecastCategory: category}
	}
	wholesale := crmTypes.CrmPipeline{Name: "Ventas mayoristas", Color: "#3b82f6", SortOrder: 1, Status: crmTypes.CrmStatusActive,
		Stages: []crmTypes.CrmPipelineStage{
			stage("Contacto", "#6b7280", 10, 10), stage("Cotización", "#0ea5e9", 30, 14), stage("Negociación", "#f59e0b", 60, 14),
			stage("Pedido de prueba", "#8b5cf6", 80, 21),
		},
		Fields: []crmTypes.CrmPipelineField{{FieldID: customFields[0].ID, Required: true}, {FieldID: customFields[2].ID}}}
	projects := crmTypes.CrmPipeline{Name: "Proyectos especiales", Color: "#10b981", SortOrder: 2, Status: crmTypes.CrmStatusActive, GoalPeriodType: crmTypes.GoalPeriodWeekly,
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

	// One team of every seller who can edit, led by the actor, and goals for each of them from a year
	// back to a few periods ahead (monthly or weekly by pipeline), so the goals grid and the forecast
	// page have data.
	team := crmTypes.CrmTeam{}
	teamPayload := crmTypes.CrmTeam{Name: "Equipo comercial", LeaderUserID: crmDataActorID, MemberUserIDs: editorIDs}
	if err := crmDataCall(crmDataActorID, "POST.crm-team", crmModule.PostCrmTeam, teamPayload, &team); err != nil {
		return err
	}
	goals := []crmTypes.CrmGoal{}
	addGoals := func(pipeline crmTypes.CrmPipeline, period int16) {
		for _, userID := range editorIDs {
			goals = append(goals,
				crmTypes.CrmGoal{PipelineID: pipeline.ID, UserID: userID, Period: period, Metric: crmTypes.CrmGoalMetricWonAmount, Target: 30_000_00},
				crmTypes.CrmGoal{PipelineID: pipeline.ID, UserID: userID, Period: period, Metric: crmTypes.CrmGoalMetricWonCount, Target: 3})
		}
	}
	// The clock is frozen at the start of the history here, so "today" comes from the real time.
	todayUnixDay := int16(realNow.Unix() / 86400)
	for _, pipeline := range pipelines {
		if pipeline.GoalPeriodType == crmTypes.GoalPeriodWeekly {
			// The projects pipeline sets weekly goals: a year back to eight weeks ahead.
			for weekOffset := -52; weekOffset <= 8; weekOffset++ {
				addGoals(pipeline, core.WeekOfDay(todayUnixDay)+int16(weekOffset))
			}
			continue
		}
		for monthOffset := -12; monthOffset <= 2; monthOffset++ {
			addGoals(pipeline, core.AddPeriods(core.PeriodOfDay(todayUnixDay), monthOffset))
		}
	}
	if err := crmDataCall(crmDataActorID, "POST.crm-goals", crmModule.PostCrmGoals, crmTypes.CrmGoalsPayload{Goals: goals}, &goals); err != nil {
		return err
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

// runCrmDataOpportunity lives one opportunity over a random span of 2 to 30 days: created on a past
// day by an Edit/Full member, moved a few stages forward inside the span (sometimes one step back and
// forward again, a renegotiated amount or a new seller), and on its last day won, lost or moved to
// the other pipeline. A span that reaches past today leaves the card open where it got to.
func runCrmDataOpportunity(random *mrand.Rand, realNow time.Time, pipeline crmTypes.CrmPipeline, pipelines []crmTypes.CrmPipeline,
	customFields []crmTypes.CrmCustomField, lostReasons []crmTypes.CrmLostReason, clientIDs, editorIDs []int32, opportunityIndex int) error {

	const day = 24 * time.Hour
	moment := realNow.AddDate(0, 0, -(1 + random.Intn(crmDataHistoryDays)))
	// The span is the days from creation to the last change: 1..29 days apart, so 2..30 calendar days.
	spanDays := 1 + random.Intn(crmDataMaxSpanDays-1)
	lastMoment := moment.Add(time.Duration(spanDays) * day)
	// Steps are short enough that walking every stage fits inside the span.
	maxStepDays := max(1, spanDays/(len(pipeline.Stages)+1))
	freezeAt := func() { core.SetHistoricalUnix(moment.Unix()) }
	// advance moves to the next change, never reaching the last day (kept for the closing) nor today.
	advance := func() bool {
		next := moment.Add(time.Duration(1+random.Intn(maxStepDays)) * day)
		if !next.Before(lastMoment) || !next.Before(realNow) {
			return false
		}
		moment = next
		return true
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
		// Around the end of its span, so the forecast of each month sees the deals closing in it.
		ExpectedCloseDate: core.TimeToFechaUnix(lastMoment) + int16(random.Intn(9)-3), CustomValues: values,
	}, &created)
	if err != nil {
		return err
	}

	moveTo := func(stageID int16) error {
		freezeAt()
		return crmDataCall(crmDataActorID, "POST.crm-opportunity-move", crmModule.PostCrmOpportunityMove,
			crmTypes.CrmOpportunityMovePayload{ID: created.ID, PipelineID: pipeline.ID, StageID: stageID}, &created)
	}
	// edit re-saves the card through the edit route; amount, owner and close date changes are logged.
	edit := func(change func(*crmTypes.CrmOpportunity)) error {
		freezeAt()
		payload := created
		change(&payload)
		return crmDataCall(crmDataActorID, "POST.crm-opportunity", crmModule.PostCrmOpportunity, payload, &created)
	}
	closeAs := func(outcome int8) error {
		freezeAt()
		payload := crmTypes.CrmOpportunityClosePayload{ID: created.ID, Outcome: outcome}
		if outcome == crmTypes.OpportunityOutcomeLost {
			payload.LostReasonID, payload.LostNote = lostReasons[random.Intn(len(lostReasons))].ID, "Registrado en la simulación"
		}
		return crmDataCall(crmDataActorID, "POST.crm-opportunity-close", crmModule.PostCrmOpportunityClose, payload, nil)
	}

	// Walk forward through the stages inside the span. Some steps go back one stage and forward again,
	// which the net-progress report must count once; some deals are renegotiated or handed over.
	reachedLastStage := true
	for stageIndex := 1; stageIndex < len(activeStages); stageIndex++ {
		if random.Intn(100) < 25 || !advance() {
			reachedLastStage = false
			break
		}
		if err := moveTo(activeStages[stageIndex].ID); err != nil {
			return err
		}
		if stageIndex >= 2 && random.Intn(100) < 20 && advance() {
			if err := moveTo(activeStages[stageIndex-1].ID); err != nil {
				return err
			}
			if !advance() {
				reachedLastStage = false
				break
			}
			if err := moveTo(activeStages[stageIndex].ID); err != nil {
				return err
			}
		}
		if random.Intn(100) < 15 && advance() {
			if err := edit(func(payload *crmTypes.CrmOpportunity) {
				payload.Amount = payload.Amount * int32(80+random.Intn(50)) / 100
			}); err != nil {
				return err
			}
		}
		if len(editorIDs) > 1 && random.Intn(100) < 6 && advance() {
			if err := edit(func(payload *crmTypes.CrmOpportunity) {
				payload.OwnerUserID = editorIDs[random.Intn(len(editorIDs))]
			}); err != nil {
				return err
			}
		}
	}

	// A few deals are lost along the way and reopened a day or two later, before their last day.
	if random.Intn(100) < 8 && advance() {
		if err := closeAs(crmTypes.OpportunityOutcomeLost); err != nil {
			return err
		}
		if !advance() {
			return nil // stays lost: the span had no room left to reopen it
		}
		if err := closeAs(crmTypes.OpportunityOutcomeOpen); err != nil {
			return err
		}
	}

	// The span reaches past today: the card is still open on the board.
	if !lastMoment.Before(realNow) {
		return nil
	}
	moment = lastMoment

	// One in ten wholesale cards crosses to the projects pipeline on its last day.
	if pipeline.ID == pipelines[0].ID && random.Intn(10) == 0 {
		freezeAt()
		return crmDataCall(crmDataActorID, "POST.crm-opportunity-move", crmModule.PostCrmOpportunityMove,
			crmTypes.CrmOpportunityMovePayload{ID: created.ID, PipelineID: pipelines[1].ID, StageID: pipelines[1].Stages[0].ID}, &created)
	}
	// Cards that got to the last stage are won more often.
	winChance := 35
	if reachedLastStage {
		winChance = 70
	}
	if random.Intn(100) < winChance {
		return closeAs(crmTypes.OpportunityOutcomeWon)
	}
	return closeAs(crmTypes.OpportunityOutcomeLost)
}
