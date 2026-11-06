package sample_records

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"app/projects"
	projectsTypes "app/projects/types"
	"app/security"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	mrand "math/rand"
	"slices"
	"strconv"
	"strings"
	"time"
)

// generate_projects_flow replays the life of agile projects through the production handlers of the
// projects module, with the clock frozen on each simulated moment: team, epics, backlog, files,
// comments, ranking, a discarded draft sprint, then sprints that are planned, started, worked card by
// card, and closed carrying their unfinished stories over. The last sprint stays active today.
//
// It is also a test of the module: after every step it reads the project back through the same GET
// routes the browser uses and checks the step's outcome and every invariant of a project
// (checkProjectInvariants). The first failure stops the run and names the step; the data stays
// behind to inspect it.

//go:embed projects_flow_catalog.json
var projectsFlowCatalogJSON []byte

const (
	// projectsFlowCreatorID creates every project: the dev login's user, so the projects open as its admin.
	projectsFlowCreatorID int32 = 1
	// projectsFlowPassword is the password of the seeded people, to log in as them in dev.
	projectsFlowPassword = "genix2026"
	// projectsFlowSequenceBits mirrors projects/types projectSequenceBits: a project record's ID is
	// ProjectID << 23 | its number inside the project.
	projectsFlowSequenceBits = 23
	projectsFlowSequenceMask = 1<<projectsFlowSequenceBits - 1
	// projectsFlowCheckAttempts: a failed check is read again before it fails the run, since the lists
	// come from materialized views that may trail the write just acknowledged.
	projectsFlowCheckAttempts = 4
)

type projectsFlowCatalog struct {
	People      []projectsFlowPerson
	Comments    []string
	StopReasons []string
	SprintGoals []string
	Projects    []projectsFlowTheme
}

type projectsFlowPerson struct {
	User      string
	FirstName string
	LastName  string
	JobTitle  string
}

type projectsFlowTheme struct {
	Name        string
	Description string
	Color       string
	Epics       []projectsFlowEpic
}

type projectsFlowEpic struct {
	Name    string
	Color   string
	Stories []projectsFlowStory
}

type projectsFlowStory struct {
	Title     string
	Summary   string
	Type      int8
	Priority  int8
	Hours     float32
	Checklist []string
}

type projectsFlowConfig struct {
	projectsCount int
	sprintsCount  int
	storiesCount  int
	sprintDays    int
	randomSeed    int64
}

type projectsFlowGenerator struct {
	config  projectsFlowConfig
	catalog projectsFlowCatalog
	random  *mrand.Rand
	// realNow anchors the timeline: the clock is frozen on simulated moments for the whole run.
	realNow time.Time
	today   int16
	// people are the seeded team, catalog order: people[0] administers every project with the creator.
	people []coreTypes.User
	// clockDay / clockMinute is the simulated moment of the last write (tick).
	clockDay     int16
	clockMinute  int
	checksPassed int
	viewRetries  int
}

// projectFlowRun is one project's simulation. snapshot is the project as read back by the last check:
// the next step plans from it.
type projectFlowRun struct {
	project         projectsTypes.Project
	adminID         int32
	memberIDs       []int32
	epicIDs         []int64
	temporaryEpicID int64
	storyIDs        []int64
	snapshot        projectSnapshot
	stats           projectFlowStats
}

type projectFlowStats struct {
	ProjectID int32
	Name      string
	Epics     int
	Stories   int
	Sprints   int
	Moves     int
	Comments  int
	Files     int
}

// projectSnapshot is a project as the browser sees it: every active record of its GET routes.
type projectSnapshot struct {
	project   projectsTypes.Project
	epics     []projectsTypes.Epic
	stories   []projectsTypes.Story
	sprints   []projectsTypes.Sprint
	documents []projectsTypes.Document
}

// projectsFlowStoryWithDetail is the body and the answer of POST.projects-stories.
type projectsFlowStoryWithDetail struct {
	Story  projectsTypes.Story
	Detail projectsTypes.StoryDetail
}

// projectsFlowCardMove is a planned move of a board card: on the work day dayIndex, to state.
type projectsFlowCardMove struct {
	dayIndex int
	state    int8
	reason   string
}

type projectsFlowHandler func(*core.HandlerArgs) core.HandlerResponse

// GenerateProjectsFlow is the entry point behind `fn-generate-projects-flow`.
func GenerateProjectsFlow(args *core.ExecArgs) core.FuncResponse {
	config, err := parseProjectsFlowArgs(args.Message)
	if err != nil {
		return args.MakeErr("Argumentos inválidos:", err)
	}
	catalog := projectsFlowCatalog{}
	if err := json.Unmarshal(projectsFlowCatalogJSON, &catalog); err != nil {
		return args.MakeErr("No se pudo leer el catálogo de proyectos:", err)
	}
	// The real clock, read once: every write below runs with the clock frozen on a simulated moment.
	realNow := time.Now()
	generator := projectsFlowGenerator{
		config: config, catalog: catalog, random: mrand.New(mrand.NewSource(config.randomSeed)),
		realNow: realNow, today: core.TimeToFechaUnix(realNow),
	}
	defer core.SetHistoricalUnix(0)
	startedAt := time.Now()
	core.Log("ProjectsFlow:: inicio", "proyectos", config.projectsCount, "sprints", config.sprintsCount,
		"historias", config.storiesCount, "semilla", config.randomSeed)

	if err := generator.ensurePeople(); err != nil {
		return args.MakeErr("No se pudo preparar al equipo:", err)
	}
	projectsStats := []projectFlowStats{}
	for projectIndex := 0; projectIndex < config.projectsCount; projectIndex++ {
		run, err := generator.runProjectFlow(projectIndex)
		if err != nil {
			return args.MakeErr(fmt.Sprintf("Falló el flujo del proyecto %d (repetir con --seed=%d):", run.project.ID, config.randomSeed), err)
		}
		projectsStats = append(projectsStats, run.stats)
	}

	core.Log("ProjectsFlow:: terminado", "chequeos", generator.checksPassed, "relecturas", generator.viewRetries)
	return core.FuncResponse{
		Message: fmt.Sprintf("%d proyectos simulados: %d chequeos correctos.", len(projectsStats), generator.checksPassed),
		Content: map[string]any{
			"projects":    projectsStats,
			"checks":      generator.checksPassed,
			"viewRetries": generator.viewRetries,
			"seed":        config.randomSeed,
			"seconds":     int(time.Since(startedAt).Seconds()),
		},
	}
}

func parseProjectsFlowArgs(message string) (projectsFlowConfig, error) {
	config := projectsFlowConfig{projectsCount: 2, sprintsCount: 4, storiesCount: 20, sprintDays: 14, randomSeed: sampleDataSeed}
	for _, argument := range strings.Fields(message) {
		name, rawValue, hasValue := strings.Cut(strings.TrimPrefix(argument, "--"), "=")
		if !hasValue {
			return config, core.Err("el argumento", argument, "necesita un valor")
		}
		parsedValue, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			return config, core.Err("el argumento", argument, "no es un número")
		}
		switch name {
		case "projects":
			config.projectsCount = int(parsedValue)
		case "sprints":
			config.sprintsCount = int(parsedValue)
		case "stories":
			config.storiesCount = int(parsedValue)
		case "sprint-days":
			config.sprintDays = int(parsedValue)
		case "seed":
			config.randomSeed = parsedValue
		default:
			return config, core.Err("argumento desconocido:", argument)
		}
	}
	switch {
	case config.projectsCount < 1 || config.projectsCount > 20:
		return config, core.Err("--projects debe estar entre 1 y 20")
	case config.sprintsCount < 1 || config.sprintsCount > 12:
		return config, core.Err("--sprints debe estar entre 1 y 12")
	case config.storiesCount < 6 || config.storiesCount > 200:
		return config, core.Err("--stories debe estar entre 6 y 200")
	case config.sprintDays < 5 || config.sprintDays > 28:
		return config, core.Err("--sprint-days debe estar entre 5 y 28")
	}
	return config, nil
}

// ensurePeople finds the catalog's people in the company, by login, and creates the missing ones with
// the projects access at the write level, which every team member needs.
func (generator *projectsFlowGenerator) ensurePeople() error {
	companyUsers := []coreTypes.User{}
	query := db.Query(&companyUsers)
	query.CompanyID.Equals(sampleCompanyID)
	if err := query.Exec(); err != nil {
		return err
	}
	for _, person := range generator.catalog.People {
		userIndex := slices.IndexFunc(companyUsers, func(user coreTypes.User) bool { return user.User == person.User })
		if userIndex >= 0 {
			generator.people = append(generator.people, companyUsers[userIndex])
			continue
		}
		newUser := coreTypes.User{
			User: person.User, FirstName: person.FirstName, LastName: person.LastName, JobTitle: person.JobTitle,
			Password:      projectsFlowPassword,
			AccesosGrants: []coreTypes.AccesoGrantRecord{{AccesoID: uint16(projectsTypes.AccesoIDProjects), Nivel: 2}},
		}
		createdUser := coreTypes.User{}
		if err := generator.call(projectsFlowCreatorID, "POST.users", security.PostUsuarios, nil, newUser, &createdUser); err != nil {
			return err
		}
		if createdUser.ID == 0 {
			return core.Err("el usuario", person.User, "no devolvió ID")
		}
		core.Log("ProjectsFlow:: usuario creado", person.User, "ID", createdUser.ID)
		generator.people = append(generator.people, createdUser)
	}
	return nil
}

// runProjectFlow simulates one project. The calendar runs backwards from today: the last sprint is
// half way through, the ones before it are closed, and the project kicks off 5 days before the first.
func (generator *projectsFlowGenerator) runProjectFlow(projectIndex int) (*projectFlowRun, error) {
	theme := generator.catalog.Projects[projectIndex%len(generator.catalog.Projects)]
	if round := projectIndex / len(generator.catalog.Projects); round > 0 {
		theme.Name = fmt.Sprintf("%s %d", theme.Name, round+1)
	}
	run := &projectFlowRun{adminID: generator.people[0].ID}
	sprintDays := generator.config.sprintDays
	sprintStartDays := make([]int16, generator.config.sprintsCount)
	lastSprintStart := generator.today - int16(sprintDays/2)
	for sprintIndex := range sprintStartDays {
		sprintStartDays[sprintIndex] = lastSprintStart - int16((generator.config.sprintsCount-1-sprintIndex)*sprintDays)
	}
	kickoffDay := sprintStartDays[0] - 5

	if err := generator.createProject(run, theme, kickoffDay, sprintStartDays); err != nil {
		return run, err
	}
	if err := generator.createEpics(run, theme); err != nil {
		return run, err
	}
	if err := generator.createStories(run, theme, kickoffDay); err != nil {
		return run, err
	}
	generator.startDay(kickoffDay + 3)
	if err := generator.attachStoryFiles(run); err != nil {
		return run, err
	}
	if err := generator.commentOnBacklog(run); err != nil {
		return run, err
	}
	if err := generator.rankStory(run); err != nil {
		return run, err
	}
	if err := generator.deleteTemporaryEpic(run); err != nil {
		return run, err
	}
	if err := generator.addProjectDocuments(run); err != nil {
		return run, err
	}
	if err := generator.discardDraftSprint(run, sprintStartDays[0]); err != nil {
		return run, err
	}
	if err := generator.runSprints(run, sprintStartDays); err != nil {
		return run, err
	}
	if err := generator.deleteBacklogStory(run); err != nil {
		return run, err
	}
	if err := generator.republishCompanyLogs(run); err != nil {
		return run, err
	}
	if err := generator.checkFinalState(run); err != nil {
		return run, err
	}
	run.stats.ProjectID, run.stats.Name = run.project.ID, run.project.Name
	run.stats.Epics, run.stats.Stories = len(run.snapshot.epics), len(run.snapshot.stories)
	core.Log("ProjectsFlow:: proyecto listo", run.project.ID, run.project.Name, "→ /projects/backlog?project="+strconv.Itoa(int(run.project.ID)))
	return run, nil
}

// createProject: the creator opens the project with the team, then its admin sets the dates once the
// sprint calendar is known.
func (generator *projectsFlowGenerator) createProject(run *projectFlowRun, theme projectsFlowTheme, kickoffDay int16, sprintStartDays []int16) error {
	generator.startDay(kickoffDay)
	members := []projectsTypes.ProjectMember{{UserID: projectsFlowCreatorID, Role: projectsTypes.ProjectRoleAdmin}}
	for personIndex, person := range generator.people {
		role := projectsTypes.ProjectRoleMember
		if personIndex == 0 {
			role = projectsTypes.ProjectRoleAdmin
		}
		members = append(members, projectsTypes.ProjectMember{UserID: person.ID, Role: role})
	}
	for _, member := range members {
		run.memberIDs = append(run.memberIDs, member.UserID)
	}
	generator.tick()
	newProject := projectsTypes.Project{Name: theme.Name, Description: theme.Description, Color: theme.Color, Members: members}
	if err := generator.call(projectsFlowCreatorID, "POST.projects-projects", projects.PostProject, nil, newProject, &run.project); err != nil {
		return err
	}
	if run.project.ID == 0 {
		return core.Err("el proyecto no devolvió ID")
	}
	core.Log("ProjectsFlow:: proyecto creado", run.project.ID, run.project.Name)
	err := generator.expect(run, "crear el proyecto", func(snapshot projectSnapshot) error {
		if snapshot.project.Name != theme.Name || len(snapshot.project.Members) != len(members) || snapshot.project.State != projectsTypes.ProjectStateActive {
			return fmt.Errorf("el proyecto guardado no coincide: %q, %d integrantes, estado %d", snapshot.project.Name, len(snapshot.project.Members), snapshot.project.State)
		}
		return nil
	})
	if err != nil {
		return err
	}

	generator.tick()
	datedProject := run.project
	datedProject.StartDate = sprintStartDays[0]
	datedProject.EndDate = sprintStartDays[len(sprintStartDays)-1] + int16(2*generator.config.sprintDays)
	if err := generator.call(run.adminID, "POST.projects-projects", projects.PostProject, nil, datedProject, &run.project); err != nil {
		return err
	}
	return generator.expect(run, "fechar el proyecto", func(snapshot projectSnapshot) error {
		if snapshot.project.StartDate != datedProject.StartDate || snapshot.project.EndDate != datedProject.EndDate {
			return fmt.Errorf("fechas %d-%d, se esperaba %d-%d", snapshot.project.StartDate, snapshot.project.EndDate, datedProject.StartDate, datedProject.EndDate)
		}
		return nil
	})
}

// createEpics saves the theme's epics plus a temporary one in one request, then renames the temporary
// one. It is deleted later, with a story in it.
func (generator *projectsFlowGenerator) createEpics(run *projectFlowRun, theme projectsFlowTheme) error {
	newEpics := []projectsTypes.Epic{}
	for _, epic := range theme.Epics {
		newEpics = append(newEpics, projectsTypes.Epic{ProjectID: run.project.ID, Name: epic.Name, Color: epic.Color})
	}
	newEpics = append(newEpics, projectsTypes.Epic{ProjectID: run.project.ID, Name: "Épica temporal", Color: "gray"})
	generator.tick()
	savedEpics := []projectsTypes.Epic{}
	if err := generator.call(generator.people[1].ID, "POST.projects-epics", projects.PostEpics, nil, newEpics, &savedEpics); err != nil {
		return err
	}
	if len(savedEpics) != len(newEpics) {
		return fmt.Errorf("se guardaron %d épicas de %d", len(savedEpics), len(newEpics))
	}
	for _, savedEpic := range savedEpics[:len(savedEpics)-1] {
		run.epicIDs = append(run.epicIDs, savedEpic.ID)
	}
	run.temporaryEpicID = savedEpics[len(savedEpics)-1].ID
	err := generator.expect(run, "crear las épicas", func(snapshot projectSnapshot) error {
		epicNumbers := []int64{}
		for _, epic := range snapshot.epics {
			epicNumbers = append(epicNumbers, epic.ID&projectsFlowSequenceMask)
		}
		slices.Sort(epicNumbers)
		for epicIndex, epicNumber := range epicNumbers {
			if epicNumber != int64(epicIndex+1) {
				return fmt.Errorf("las épicas no se numeran 1..%d en el proyecto: %v", len(newEpics), epicNumbers)
			}
		}
		if len(epicNumbers) != len(newEpics) {
			return fmt.Errorf("hay %d épicas, se esperaban %d", len(epicNumbers), len(newEpics))
		}
		return nil
	})
	if err != nil {
		return err
	}

	renamedEpic := savedEpics[len(savedEpics)-1]
	renamedEpic.Name = "Épica temporal (por eliminar)"
	generator.tick()
	if err := generator.call(generator.people[2].ID, "POST.projects-epics", projects.PostEpics, nil, []projectsTypes.Epic{renamedEpic}, nil); err != nil {
		return err
	}
	return generator.expect(run, "renombrar una épica", func(snapshot projectSnapshot) error {
		epicIndex := slices.IndexFunc(snapshot.epics, func(epic projectsTypes.Epic) bool { return epic.ID == renamedEpic.ID })
		if epicIndex < 0 || snapshot.epics[epicIndex].Name != renamedEpic.Name {
			return fmt.Errorf("la épica %d no quedó renombrada", renamedEpic.ID)
		}
		return nil
	})
}

// createStories writes the backlog over the three days after the kickoff, cycling the theme's stories
// when more are asked for. Some are assigned on creation and some carry a description and checklist.
// The second story goes in the temporary epic.
func (generator *projectsFlowGenerator) createStories(run *projectFlowRun, theme projectsFlowTheme, kickoffDay int16) error {
	type themeStory struct {
		story     projectsFlowStory
		epicIndex int
	}
	themeStories := []themeStory{}
	for epicIndex, epic := range theme.Epics {
		for _, story := range epic.Stories {
			themeStories = append(themeStories, themeStory{story: story, epicIndex: epicIndex})
		}
	}
	storiesPerDay := (generator.config.storiesCount + 2) / 3
	previousRank := 0.0
	for storyIndex := 0; storyIndex < generator.config.storiesCount; storyIndex++ {
		if storyIndex%storiesPerDay == 0 {
			generator.startDay(kickoffDay + int16(storyIndex/storiesPerDay))
		}
		seed := themeStories[storyIndex%len(themeStories)]
		title := seed.story.Title
		if round := storyIndex / len(themeStories); round > 0 {
			title = fmt.Sprintf("%s (parte %d)", title, round+1)
		}
		newStory := projectsTypes.Story{
			ProjectID: run.project.ID, Type: seed.story.Type, Title: title, Summary: seed.story.Summary,
			EpicID: run.epicIDs[seed.epicIndex], Priority: seed.story.Priority, EstimatedHours: seed.story.Hours,
		}
		if storyIndex == 1 {
			newStory.EpicID = run.temporaryEpicID
		}
		if generator.random.Float64() < 0.3 {
			newStory.AssigneeID = generator.pickMember(run)
		}
		newDetail := projectsTypes.StoryDetail{}
		if len(seed.story.Checklist) > 0 || generator.random.Float64() < 0.4 {
			newDetail.Description = "<p>" + html.EscapeString(seed.story.Summary) + "</p>"
			for _, checklistText := range seed.story.Checklist {
				newDetail.Checklist = append(newDetail.Checklist, projectsTypes.ChecklistItem{Text: checklistText})
			}
		}

		generator.tick()
		savedStory := projectsFlowStoryWithDetail{}
		storyBody := projectsFlowStoryWithDetail{Story: newStory, Detail: newDetail}
		if err := generator.call(generator.pickMember(run), "POST.projects-stories", projects.PostStory, nil, storyBody, &savedStory); err != nil {
			return err
		}
		storyID := savedStory.Story.ID
		run.storyIDs = append(run.storyIDs, storyID)
		expectedNumber := int32(storyIndex + 1)
		err := generator.expect(run, fmt.Sprintf("crear la historia H%02d", expectedNumber), func(snapshot projectSnapshot) error {
			story, isFound := findSnapshotStory(snapshot, storyID)
			if !isFound {
				return fmt.Errorf("la historia %d no está en la lista", storyID)
			}
			if projectsTypes.StoryNumber(story.ID) != expectedNumber {
				return fmt.Errorf("la historia quedó numerada H%d, se esperaba H%d", projectsTypes.StoryNumber(story.ID), expectedNumber)
			}
			if story.Title != title || story.AssigneeID != newStory.AssigneeID || story.State != projectsTypes.StoryStateNotPlanned {
				return fmt.Errorf("la historia guardada no coincide: %q, responsable %d, estado %d", story.Title, story.AssigneeID, story.State)
			}
			if story.BacklogRank <= previousRank {
				return fmt.Errorf("la historia nueva no quedó última en el backlog: rango %v tras %v", story.BacklogRank, previousRank)
			}
			return nil
		})
		if err != nil {
			return err
		}
		previousStory, _ := findSnapshotStory(run.snapshot, storyID)
		previousRank = previousStory.BacklogRank
		if newStory.AssigneeID != 0 {
			if err := generator.expectAssignment(run, newStory.AssigneeID, storyID, true); err != nil {
				return err
			}
		}
	}
	run.stats.Stories = len(run.storyIDs)
	return nil
}

// attachStoryFiles uploads two files to the first story, saves them on it, then removes one: the
// Documents page must follow each save.
func (generator *projectsFlowGenerator) attachStoryFiles(run *projectFlowRun) error {
	storyID := run.storyIDs[0]
	story, _ := findSnapshotStory(run.snapshot, storyID)
	uploader := generator.pickMember(run)
	specification, err := generator.uploadFile(run, uploader, "especificacion-funcional.txt", "text/plain", projectsFlowSpecificationText(story.Title))
	if err != nil {
		return err
	}
	mockup, err := generator.uploadFile(run, uploader, "maqueta-pantalla.svg", "image/svg+xml", projectsFlowMockupSVG(story.Title))
	if err != nil {
		return err
	}
	if _, err := generator.editStory(run, uploader, storyID, func(story *projectsTypes.Story, detail *projectsTypes.StoryDetail) {
		detail.Attachments = append(detail.Attachments, specification, mockup)
	}); err != nil {
		return err
	}
	if err := generator.expectStoryFiles(run, "adjuntar dos archivos a una historia", storyID, []string{specification.Key, mockup.Key}); err != nil {
		return err
	}

	if _, err := generator.editStory(run, uploader, storyID, func(story *projectsTypes.Story, detail *projectsTypes.StoryDetail) {
		detail.Attachments = slices.DeleteFunc(detail.Attachments, func(attachment projectsTypes.Attachment) bool { return attachment.Key == mockup.Key })
	}); err != nil {
		return err
	}
	return generator.expectStoryFiles(run, "quitar un archivo de una historia", storyID, []string{specification.Key})
}

// expectStoryFiles checks a story's own files: its counter, its detail, and its rows on the Documents
// page, each linked from the attachment by DocumentID.
func (generator *projectsFlowGenerator) expectStoryFiles(run *projectFlowRun, step string, storyID int64, expectedKeys []string) error {
	err := generator.expect(run, step, func(snapshot projectSnapshot) error {
		story, _ := findSnapshotStory(snapshot, storyID)
		if int(story.AttachmentsCount) != len(expectedKeys) {
			return fmt.Errorf("la historia cuenta %d archivos, se esperaban %d", story.AttachmentsCount, len(expectedKeys))
		}
		storyDocumentKeys := []string{}
		for _, document := range snapshot.documents {
			if document.StoryID == storyID && document.ActivityID == 0 {
				storyDocumentKeys = append(storyDocumentKeys, document.Key)
			}
		}
		if !sameKeys(storyDocumentKeys, expectedKeys) {
			return fmt.Errorf("Documentos lista %v de la historia, se esperaba %v", storyDocumentKeys, expectedKeys)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return generator.checkWithRetries(step+" (detalle)", func() error {
		detail, err := generator.readStoryDetail(storyID)
		if err != nil {
			return err
		}
		detailKeys := []string{}
		for _, attachment := range detail.Attachments {
			detailKeys = append(detailKeys, attachment.Key)
			if err := checkAttachmentDocument(run.snapshot, attachment, storyID, 0); err != nil {
				return err
			}
		}
		if !sameKeys(detailKeys, expectedKeys) {
			return fmt.Errorf("el detalle tiene %v, se esperaba %v", detailKeys, expectedKeys)
		}
		return nil
	})
}

// commentOnBacklog comments on four stories; the last comment carries a file.
func (generator *projectsFlowGenerator) commentOnBacklog(run *projectFlowRun) error {
	for commentIndex := 0; commentIndex < 4; commentIndex++ {
		storyID := run.storyIDs[generator.random.Intn(len(run.storyIDs))]
		if err := generator.addComment(run, storyID, generator.pickMember(run), commentIndex == 3); err != nil {
			return err
		}
	}
	return nil
}

// addComment posts a comment and checks the story's counter, the timeline, and, with a file, its row
// on the Documents page linked to the comment.
func (generator *projectsFlowGenerator) addComment(run *projectFlowRun, storyID int64, authorID int32, withFile bool) error {
	story, _ := findSnapshotStory(run.snapshot, storyID)
	attachments := []projectsTypes.Attachment{}
	if withFile {
		evidence, err := generator.uploadFile(run, authorID, "casos-de-prueba.csv", "text/csv", projectsFlowTestCasesCSV(story.Title))
		if err != nil {
			return err
		}
		attachments = append(attachments, evidence)
	}
	commentText := generator.catalog.Comments[generator.random.Intn(len(generator.catalog.Comments))]
	generator.tick()
	savedComment := projectsTypes.StoryActivity{}
	commentBody := map[string]any{"ProjectID": run.project.ID, "StoryID": storyID, "Text": commentText, "Attachments": attachments}
	if err := generator.call(authorID, "POST.projects-story-comments", projects.PostStoryComment, nil, commentBody, &savedComment); err != nil {
		return err
	}
	run.stats.Comments++
	step := fmt.Sprintf("comentar la historia H%02d", projectsTypes.StoryNumber(storyID))
	err := generator.expect(run, step, func(snapshot projectSnapshot) error {
		commentedStory, _ := findSnapshotStory(snapshot, storyID)
		if commentedStory.CommentsCount != story.CommentsCount+1 {
			return fmt.Errorf("la historia cuenta %d comentarios, se esperaban %d", commentedStory.CommentsCount, story.CommentsCount+1)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return generator.checkWithRetries(step+" (actividad)", func() error {
		activities, err := generator.readStoryActivity(storyID)
		if err != nil {
			return err
		}
		activityIndex := slices.IndexFunc(activities, func(activity projectsTypes.StoryActivity) bool { return activity.ID == savedComment.ID })
		if activityIndex < 0 {
			return fmt.Errorf("el comentario %d no está en la actividad", savedComment.ID)
		}
		comment := activities[activityIndex]
		if comment.Kind != projectsTypes.ActivityKindComment || comment.Text != commentText || comment.AuthorID != authorID || len(comment.Attachments) != len(attachments) {
			return fmt.Errorf("el comentario guardado no coincide: tipo %d, %q, autor %d, %d archivos", comment.Kind, comment.Text, comment.AuthorID, len(comment.Attachments))
		}
		for _, attachment := range comment.Attachments {
			if err := checkAttachmentDocument(run.snapshot, attachment, storyID, comment.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

// rankStory drags the last story of the backlog between the first two, into the first one's epic.
func (generator *projectsFlowGenerator) rankStory(run *projectFlowRun) error {
	backlog := slices.Clone(run.snapshot.stories)
	slices.SortFunc(backlog, func(left, right projectsTypes.Story) int { return compareFloat(left.BacklogRank, right.BacklogRank) })
	firstStory, secondStory, movedStory := backlog[0], backlog[1], backlog[len(backlog)-1]
	generator.tick()
	rankBody := map[string]any{
		"ProjectID": run.project.ID, "StoryID": movedStory.ID, "PreviousStoryID": firstStory.ID, "NextStoryID": secondStory.ID, "EpicID": firstStory.EpicID,
	}
	if err := generator.call(generator.pickMember(run), "POST.projects-stories-rank", projects.PostStoryRank, nil, rankBody, nil); err != nil {
		return err
	}
	return generator.expect(run, "reordenar el backlog", func(snapshot projectSnapshot) error {
		story, _ := findSnapshotStory(snapshot, movedStory.ID)
		if story.BacklogRank <= firstStory.BacklogRank || story.BacklogRank >= secondStory.BacklogRank || story.EpicID != firstStory.EpicID {
			return fmt.Errorf("la historia movida quedó con rango %v (entre %v y %v) y épica %d (se esperaba %d)",
				story.BacklogRank, firstStory.BacklogRank, secondStory.BacklogRank, story.EpicID, firstStory.EpicID)
		}
		return nil
	})
}

// deleteTemporaryEpic deletes the temporary epic (admins): its story stays, without an epic.
func (generator *projectsFlowGenerator) deleteTemporaryEpic(run *projectFlowRun) error {
	generator.tick()
	deletedEpic := []projectsTypes.Epic{{ID: run.temporaryEpicID, ProjectID: run.project.ID, Status: projectsTypes.StatusDeleted}}
	if err := generator.call(run.adminID, "POST.projects-epics", projects.PostEpics, nil, deletedEpic, nil); err != nil {
		return err
	}
	return generator.expect(run, "eliminar una épica con historias", func(snapshot projectSnapshot) error {
		if slices.ContainsFunc(snapshot.epics, func(epic projectsTypes.Epic) bool { return epic.ID == run.temporaryEpicID }) {
			return fmt.Errorf("la épica %d sigue activa", run.temporaryEpicID)
		}
		if story, _ := findSnapshotStory(snapshot, run.storyIDs[1]); story.EpicID != 0 {
			return fmt.Errorf("la historia de la épica eliminada sigue en la épica %d", story.EpicID)
		}
		return nil
	})
}

// addProjectDocuments adds two documents to the project itself and deletes the second.
func (generator *projectsFlowGenerator) addProjectDocuments(run *projectFlowRun) error {
	addDocument := func(name string, contentType string, content string) (projectsTypes.Document, error) {
		uploaded, err := generator.uploadFile(run, generator.pickMember(run), name, contentType, content)
		if err != nil {
			return projectsTypes.Document{}, err
		}
		generator.tick()
		savedDocument := projectsTypes.Document{}
		newDocument := projectsTypes.Document{
			ProjectID: run.project.ID, Key: uploaded.Key, Name: uploaded.Name, ContentType: uploaded.ContentType, Size: uploaded.Size, Status: projectsTypes.StatusActive,
		}
		err = generator.call(generator.pickMember(run), "POST.projects-documents", projects.PostDocument, nil, newDocument, &savedDocument)
		return savedDocument, err
	}
	charter, err := addDocument("acta-de-constitucion.txt", "text/plain", projectsFlowCharterText(run.project))
	if err != nil {
		return err
	}
	schedule, err := addDocument("borrador-cronograma.csv", "text/csv", "Hito,Fecha\nInicio,"+projectsFlowDate(run.project.StartDate)+"\n")
	if err != nil {
		return err
	}
	if err := generator.expectProjectDocuments(run, "agregar documentos al proyecto", []string{charter.Key, schedule.Key}); err != nil {
		return err
	}
	generator.tick()
	deletedDocument := projectsTypes.Document{ID: schedule.ID, ProjectID: run.project.ID, Status: projectsTypes.StatusDeleted}
	if err := generator.call(run.adminID, "POST.projects-documents", projects.PostDocument, nil, deletedDocument, nil); err != nil {
		return err
	}
	return generator.expectProjectDocuments(run, "eliminar un documento del proyecto", []string{charter.Key})
}

func (generator *projectsFlowGenerator) expectProjectDocuments(run *projectFlowRun, step string, expectedKeys []string) error {
	return generator.expect(run, step, func(snapshot projectSnapshot) error {
		projectDocumentKeys := []string{}
		for _, document := range snapshot.documents {
			if document.StoryID == 0 {
				projectDocumentKeys = append(projectDocumentKeys, document.Key)
			}
		}
		if !sameKeys(projectDocumentKeys, expectedKeys) {
			return fmt.Errorf("los documentos del proyecto son %v, se esperaba %v", projectDocumentKeys, expectedKeys)
		}
		return nil
	})
}

// discardDraftSprint plans a sprint with one story and deletes it while still planned: the story must
// go back to not planned, with no sprint.
func (generator *projectsFlowGenerator) discardDraftSprint(run *projectFlowRun, firstSprintStart int16) error {
	storyID := run.storyIDs[len(run.storyIDs)-1]
	generator.tick()
	draftSprint := projectsTypes.Sprint{}
	draftBody := map[string]any{
		"Sprint":   projectsTypes.Sprint{ProjectID: run.project.ID, Name: "Sprint borrador", StartDate: firstSprintStart, EndDate: firstSprintStart + int16(generator.config.sprintDays-1)},
		"StoryIDs": []int64{storyID},
	}
	if err := generator.call(run.adminID, "POST.projects-sprints", projects.PostSprint, nil, draftBody, &draftSprint); err != nil {
		return err
	}
	err := generator.expect(run, "planificar un sprint borrador", func(snapshot projectSnapshot) error {
		story, _ := findSnapshotStory(snapshot, storyID)
		if story.State != projectsTypes.StoryStatePending || story.LastSprintID != draftSprint.ID {
			return fmt.Errorf("la historia planificada quedó en estado %d y sprint %d", story.State, story.LastSprintID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	generator.tick()
	deletedSprint := map[string]any{"Sprint": projectsTypes.Sprint{ID: draftSprint.ID, ProjectID: run.project.ID, Status: projectsTypes.StatusDeleted}}
	if err := generator.call(run.adminID, "POST.projects-sprints", projects.PostSprint, nil, deletedSprint, nil); err != nil {
		return err
	}
	return generator.expect(run, "eliminar el sprint borrador", func(snapshot projectSnapshot) error {
		if slices.ContainsFunc(snapshot.sprints, func(sprint projectsTypes.Sprint) bool { return sprint.ID == draftSprint.ID }) {
			return fmt.Errorf("el sprint borrador %d sigue activo", draftSprint.ID)
		}
		story, _ := findSnapshotStory(snapshot, storyID)
		if story.State != projectsTypes.StoryStateNotPlanned || story.LastSprintID != 0 || len(story.SprintIDs) != 0 {
			return fmt.Errorf("la historia del borrador quedó en estado %d, sprint %d, sprints %v", story.State, story.LastSprintID, story.SprintIDs)
		}
		return nil
	})
}

// runSprints plans, starts and works each sprint. A sprint's last day plans the next one, then closes
// it carrying its unfinished stories there (one goes back to the backlog). The last sprint is worked
// up to yesterday and stays active.
func (generator *projectsFlowGenerator) runSprints(run *projectFlowRun, sprintStartDays []int16) error {
	plannedStoriesCount := generator.config.storiesCount * 3 / 4
	batchSize := max(1, (plannedStoriesCount+len(sprintStartDays)-1)/len(sprintStartDays))
	previousSprintID := int64(0)
	for sprintIndex, sprintStart := range sprintStartDays {
		generator.startDay(sprintStart - 1)
		sprint, err := generator.planSprint(run, sprintIndex, sprintStart, batchSize)
		if err != nil {
			return err
		}
		if previousSprintID != 0 {
			if err := generator.closeSprint(run, previousSprintID, sprint.ID); err != nil {
				return err
			}
		}
		generator.startDay(sprintStart)
		if err := generator.startSprint(run, sprint.ID); err != nil {
			return err
		}
		lastWorkDay := min(sprintStart+int16(generator.config.sprintDays-2), generator.today-1)
		if err := generator.workSprint(run, sprint.ID, sprintStart, lastWorkDay); err != nil {
			return err
		}
		if sprintIndex == min(1, len(sprintStartDays)-1) {
			if err := generator.reassignSprintStory(run, sprint.ID); err != nil {
				return err
			}
		}
		previousSprintID = sprint.ID
		run.stats.Sprints++
	}
	return nil
}

// planSprint creates the sprint with the next stories of the backlog, assigning the unassigned ones.
func (generator *projectsFlowGenerator) planSprint(run *projectFlowRun, sprintIndex int, sprintStart int16, batchSize int) (projectsTypes.Sprint, error) {
	storiesInOpenSprints := map[int64]bool{}
	for _, sprint := range run.snapshot.sprints {
		if sprint.State != projectsTypes.SprintStateClosed {
			for _, sprintStory := range sprint.Stories {
				storiesInOpenSprints[sprintStory.StoryID] = true
			}
		}
	}
	backlog := slices.DeleteFunc(slices.Clone(run.snapshot.stories), func(story projectsTypes.Story) bool {
		return story.State == projectsTypes.StoryStateFinished || storiesInOpenSprints[story.ID]
	})
	slices.SortFunc(backlog, func(left, right projectsTypes.Story) int { return compareFloat(left.BacklogRank, right.BacklogRank) })
	plannedStoryIDs := []int64{}
	storyAssignees := []map[string]any{}
	expectedAssignees := map[int64]int32{}
	for _, story := range backlog[:min(batchSize, len(backlog))] {
		plannedStoryIDs = append(plannedStoryIDs, story.ID)
		expectedAssignees[story.ID] = story.AssigneeID
		if story.AssigneeID == 0 {
			assigneeID := generator.pickMember(run)
			storyAssignees = append(storyAssignees, map[string]any{"StoryID": story.ID, "AssigneeID": assigneeID})
			expectedAssignees[story.ID] = assigneeID
		}
	}

	generator.tick()
	sprintName := fmt.Sprintf("Sprint %d", sprintIndex+1)
	newSprint := projectsTypes.Sprint{
		ProjectID: run.project.ID, Name: sprintName, Goal: generator.catalog.SprintGoals[sprintIndex%len(generator.catalog.SprintGoals)],
		StartDate: sprintStart, EndDate: sprintStart + int16(generator.config.sprintDays-1),
	}
	savedSprint := projectsTypes.Sprint{}
	sprintBody := map[string]any{"Sprint": newSprint, "StoryIDs": plannedStoryIDs, "StoryAssignees": storyAssignees}
	if err := generator.call(run.adminID, "POST.projects-sprints", projects.PostSprint, nil, sprintBody, &savedSprint); err != nil {
		return savedSprint, err
	}
	err := generator.expect(run, "planificar "+sprintName, func(snapshot projectSnapshot) error {
		sprint, isFound := findSnapshotSprint(snapshot, savedSprint.ID)
		if !isFound || sprint.State != projectsTypes.SprintStatePlanned || sprint.Name != sprintName || len(sprint.Stories) != len(plannedStoryIDs) {
			return fmt.Errorf("el sprint planificado no coincide: encontrado %v, estado %d, %d historias de %d", isFound, sprint.State, len(sprint.Stories), len(plannedStoryIDs))
		}
		for _, storyID := range plannedStoryIDs {
			story, _ := findSnapshotStory(snapshot, storyID)
			if story.State != projectsTypes.StoryStatePending || story.LastSprintID != sprint.ID || story.AssigneeID != expectedAssignees[storyID] {
				return fmt.Errorf("la historia H%02d quedó en estado %d, sprint %d, responsable %d (se esperaba %d)",
					projectsTypes.StoryNumber(storyID), story.State, story.LastSprintID, story.AssigneeID, expectedAssignees[storyID])
			}
		}
		return nil
	})
	if err != nil {
		return savedSprint, err
	}
	for _, storyAssignee := range storyAssignees {
		if err := generator.expectAssignment(run, storyAssignee["AssigneeID"].(int32), storyAssignee["StoryID"].(int64), true); err != nil {
			return savedSprint, err
		}
	}
	return savedSprint, nil
}

// closeSprint closes the sprint: its unfinished stories go on in the next sprint, keeping their state,
// except that the first one usually goes back to the backlog when the sprint held several.
func (generator *projectsFlowGenerator) closeSprint(run *projectFlowRun, closingSprintID int64, nextSprintID int64) error {
	closingSprint, _ := findSnapshotSprint(run.snapshot, closingSprintID)
	carryOvers := []projects.SprintCarryOver{}
	stateAtClose := map[int64]int8{}
	for _, sprintStory := range closingSprint.Stories {
		if sprintStory.State == projectsTypes.StoryStateFinished {
			continue
		}
		carryOver := projects.SprintCarryOver{StoryID: sprintStory.StoryID, ToSprintID: nextSprintID}
		if len(carryOvers) == 0 && len(closingSprint.Stories) > 1 && generator.random.Float64() < 0.7 {
			carryOver.ToSprintID = 0
		}
		carryOvers = append(carryOvers, carryOver)
		stateAtClose[sprintStory.StoryID] = sprintStory.State
	}
	generator.tick()
	closeBody := map[string]any{"ProjectID": run.project.ID, "SprintID": closingSprintID, "CarryOvers": carryOvers}
	if err := generator.call(run.adminID, "POST.projects-sprint-close", projects.PostSprintClose, nil, closeBody, nil); err != nil {
		return err
	}
	return generator.expect(run, "cerrar "+closingSprint.Name, func(snapshot projectSnapshot) error {
		closedSprint, _ := findSnapshotSprint(snapshot, closingSprintID)
		nextSprint, _ := findSnapshotSprint(snapshot, nextSprintID)
		if closedSprint.State != projectsTypes.SprintStateClosed {
			return fmt.Errorf("el sprint quedó en estado %d", closedSprint.State)
		}
		for _, carryOver := range carryOvers {
			storyNumber := projectsTypes.StoryNumber(carryOver.StoryID)
			closedPosition := sprintStoryPosition(closedSprint, carryOver.StoryID)
			if closedPosition < 0 {
				return fmt.Errorf("H%02d ya no está en el sprint cerrado", storyNumber)
			}
			if closedSprintStory := closedSprint.Stories[closedPosition]; closedSprintStory.CarriedToSprintID != carryOver.ToSprintID {
				return fmt.Errorf("H%02d quedó trasladada a %d, se esperaba %d", storyNumber, closedSprintStory.CarriedToSprintID, carryOver.ToSprintID)
			}
			story, _ := findSnapshotStory(snapshot, carryOver.StoryID)
			if carryOver.ToSprintID == 0 {
				if story.LastSprintID != closingSprintID {
					return fmt.Errorf("H%02d volvió al backlog pero su último sprint es %d", storyNumber, story.LastSprintID)
				}
				continue
			}
			nextPosition := sprintStoryPosition(nextSprint, carryOver.StoryID)
			if nextPosition < 0 || nextSprint.Stories[nextPosition].State != stateAtClose[carryOver.StoryID] || story.LastSprintID != nextSprintID {
				return fmt.Errorf("H%02d no siguió en el sprint siguiente con su estado %d", storyNumber, stateAtClose[carryOver.StoryID])
			}
		}
		return nil
	})
}

func (generator *projectsFlowGenerator) startSprint(run *projectFlowRun, sprintID int64) error {
	generator.tick()
	startBody := map[string]any{"ProjectID": run.project.ID, "SprintID": sprintID}
	if err := generator.call(run.adminID, "POST.projects-sprint-start", projects.PostSprintStart, nil, startBody, nil); err != nil {
		return err
	}
	return generator.expect(run, "iniciar el sprint", func(snapshot projectSnapshot) error {
		if sprint, _ := findSnapshotSprint(snapshot, sprintID); sprint.State != projectsTypes.SprintStateActive {
			return fmt.Errorf("el sprint quedó en estado %d", sprint.State)
		}
		return nil
	})
}

// workSprint walks the sprint's cards day by day over its work days (Monday to Friday), each moved by
// its assignee, with the odd comment.
func (generator *projectsFlowGenerator) workSprint(run *projectFlowRun, sprintID int64, fromDay int16, toDay int16) error {
	workDays := []int16{}
	for day := fromDay; day <= toDay; day++ {
		if weekday := generator.dayStart(day).Weekday(); weekday != time.Saturday && weekday != time.Sunday {
			workDays = append(workDays, day)
		}
	}
	if len(workDays) == 0 {
		return nil
	}
	sprint, _ := findSnapshotSprint(run.snapshot, sprintID)
	movesByStory := map[int64][]projectsFlowCardMove{}
	for _, sprintStory := range sprint.Stories {
		movesByStory[sprintStory.StoryID] = generator.planCardMoves(sprintStory.State, len(workDays))
	}
	for dayIndex, day := range workDays {
		generator.startDay(day)
		for _, sprintStory := range sprint.Stories {
			for _, cardMove := range movesByStory[sprintStory.StoryID] {
				if cardMove.dayIndex != dayIndex {
					continue
				}
				if err := generator.moveCard(run, sprintID, sprintStory.StoryID, cardMove); err != nil {
					return err
				}
			}
		}
		if len(sprint.Stories) > 0 && generator.random.Float64() < 0.3 {
			commentedStory := sprint.Stories[generator.random.Intn(len(sprint.Stories))]
			if err := generator.addComment(run, commentedStory.StoryID, generator.storyWorker(run, commentedStory.StoryID), false); err != nil {
				return err
			}
		}
	}
	return nil
}

// planCardMoves: a card walks Pending → In development → In review → Finished, one step every 1-3
// work days from a random start in the first half of the sprint; one in six stops on the way and
// resumes the next day. What doesn't fit in the work days stays unfinished, to be carried over.
func (generator *projectsFlowGenerator) planCardMoves(currentState int8, workDaysCount int) []projectsFlowCardMove {
	cardMoves := []projectsFlowCardMove{}
	dayIndex := generator.random.Intn(max(1, workDaysCount/2))
	state := currentState
	moveTo := func(toState int8, reason string, nextDayGap int) {
		cardMoves = append(cardMoves, projectsFlowCardMove{dayIndex: dayIndex, state: toState, reason: reason})
		state = toState
		dayIndex += nextDayGap
	}
	if state == projectsTypes.StoryStatePending || state == projectsTypes.StoryStateStopped {
		moveTo(projectsTypes.StoryStateInDevelopment, "", 1+generator.random.Intn(3))
	}
	if state == projectsTypes.StoryStateInDevelopment && generator.random.Intn(6) == 0 {
		moveTo(projectsTypes.StoryStateStopped, generator.catalog.StopReasons[generator.random.Intn(len(generator.catalog.StopReasons))], 1)
		moveTo(projectsTypes.StoryStateInDevelopment, "", 1+generator.random.Intn(2))
	}
	if state == projectsTypes.StoryStateInDevelopment {
		moveTo(projectsTypes.StoryStateInReview, "", generator.random.Intn(2))
	}
	if state == projectsTypes.StoryStateInReview {
		moveTo(projectsTypes.StoryStateFinished, "", 0)
	}
	return slices.DeleteFunc(cardMoves, func(cardMove projectsFlowCardMove) bool { return cardMove.dayIndex >= workDaysCount })
}

// moveCard drops a card at the end of its target column and checks the board, the story's final state
// with the day it entered it, and the history line.
func (generator *projectsFlowGenerator) moveCard(run *projectFlowRun, sprintID int64, storyID int64, cardMove projectsFlowCardMove) error {
	sprint, _ := findSnapshotSprint(run.snapshot, sprintID)
	storyPosition := sprintStoryPosition(sprint, storyID)
	if storyPosition < 0 {
		return fmt.Errorf("H%02d ya no está en %s", projectsTypes.StoryNumber(storyID), sprint.Name)
	}
	fromState := sprint.Stories[storyPosition].State
	previousStoryID, previousRank := int64(0), 0.0
	for _, sprintStory := range sprint.Stories {
		if sprintStory.State == cardMove.state && sprintStory.StoryID != storyID && (previousStoryID == 0 || sprintStory.Rank > previousRank) {
			previousStoryID, previousRank = sprintStory.StoryID, sprintStory.Rank
		}
	}
	workerID := generator.storyWorker(run, storyID)
	generator.tick()
	moveBody := map[string]any{
		"ProjectID": run.project.ID, "SprintID": sprintID, "StoryID": storyID, "State": cardMove.state,
		"PreviousStoryID": previousStoryID, "NextStoryID": 0, "Reason": cardMove.reason,
	}
	if err := generator.call(workerID, "POST.projects-sprint-move", projects.PostSprintMove, nil, moveBody, nil); err != nil {
		return err
	}
	run.stats.Moves++
	movedDay := generator.clockDay
	step := fmt.Sprintf("mover H%02d de %d a %d en %s", projectsTypes.StoryNumber(storyID), fromState, cardMove.state, sprint.Name)
	err := generator.expect(run, step, func(snapshot projectSnapshot) error {
		movedSprint, _ := findSnapshotSprint(snapshot, sprintID)
		movedPosition := sprintStoryPosition(movedSprint, storyID)
		if movedPosition < 0 {
			return fmt.Errorf("la tarjeta ya no está en el sprint")
		}
		sprintStory := movedSprint.Stories[movedPosition]
		if sprintStory.State != cardMove.state || (previousStoryID != 0 && sprintStory.Rank <= previousRank) {
			return fmt.Errorf("la tarjeta quedó en estado %d con rango %v (tras %v)", sprintStory.State, sprintStory.Rank, previousRank)
		}
		story, _ := findSnapshotStory(snapshot, storyID)
		if story.State != cardMove.state {
			return fmt.Errorf("la historia quedó en estado %d", story.State)
		}
		stateDateIndex := slices.IndexFunc(story.StateDates, func(stateDate projectsTypes.StoryStateDate) bool { return stateDate.State == cardMove.state })
		if fromState != cardMove.state && (stateDateIndex < 0 || story.StateDates[stateDateIndex].Date != movedDay) {
			return fmt.Errorf("StateDates no registra el estado %d el día %d: %v", cardMove.state, movedDay, story.StateDates)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return generator.checkWithRetries(step+" (historial)", func() error {
		activities, err := generator.readStoryActivity(storyID)
		if err != nil {
			return err
		}
		lastStateChange := projectsTypes.StoryActivity{}
		for _, activity := range activities {
			if activity.Kind == projectsTypes.ActivityKindStateChange && activity.ID > lastStateChange.ID {
				lastStateChange = activity
			}
		}
		if lastStateChange.FromState != fromState || lastStateChange.ToState != cardMove.state || lastStateChange.SprintID != sprintID || lastStateChange.Text != cardMove.reason {
			return fmt.Errorf("el último cambio de estado del historial es %d→%d en %d (%q)", lastStateChange.FromState, lastStateChange.ToState, lastStateChange.SprintID, lastStateChange.Text)
		}
		return nil
	})
}

// reassignSprintStory hands an unfinished card of the sprint to another member, from the story form:
// the story leaves its old assignee's list and enters the new one's.
func (generator *projectsFlowGenerator) reassignSprintStory(run *projectFlowRun, sprintID int64) error {
	sprint, _ := findSnapshotSprint(run.snapshot, sprintID)
	storyIndex := slices.IndexFunc(sprint.Stories, func(sprintStory projectsTypes.SprintStory) bool {
		return sprintStory.State != projectsTypes.StoryStateFinished
	})
	if storyIndex < 0 {
		return nil
	}
	storyID := sprint.Stories[storyIndex].StoryID
	story, _ := findSnapshotStory(run.snapshot, storyID)
	previousAssigneeID := story.AssigneeID
	newAssigneeID := previousAssigneeID
	for newAssigneeID == previousAssigneeID {
		newAssigneeID = generator.pickMember(run)
	}
	if _, err := generator.editStory(run, run.adminID, storyID, func(story *projectsTypes.Story, detail *projectsTypes.StoryDetail) {
		story.AssigneeID = newAssigneeID
	}); err != nil {
		return err
	}
	err := generator.expect(run, fmt.Sprintf("reasignar H%02d", projectsTypes.StoryNumber(storyID)), func(snapshot projectSnapshot) error {
		if story, _ := findSnapshotStory(snapshot, storyID); story.AssigneeID != newAssigneeID {
			return fmt.Errorf("la historia quedó con el responsable %d, se esperaba %d", story.AssigneeID, newAssigneeID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := generator.expectAssignment(run, newAssigneeID, storyID, true); err != nil {
		return err
	}
	if previousAssigneeID != 0 {
		return generator.expectAssignment(run, previousAssigneeID, storyID, false)
	}
	return nil
}

// deleteBacklogStory deletes a story that was never planned (admins), an assigned one when there is:
// it leaves the backlog and its assignee's list.
func (generator *projectsFlowGenerator) deleteBacklogStory(run *projectFlowRun) error {
	candidates := slices.DeleteFunc(slices.Clone(run.snapshot.stories), func(story projectsTypes.Story) bool { return story.LastSprintID != 0 })
	if len(candidates) == 0 {
		return nil
	}
	slices.SortFunc(candidates, func(left, right projectsTypes.Story) int { return int(right.AssigneeID) - int(left.AssigneeID) })
	deletedStory := candidates[0]
	generator.tick()
	deleteBody := projectsFlowStoryWithDetail{Story: projectsTypes.Story{ID: deletedStory.ID, ProjectID: run.project.ID, Status: projectsTypes.StatusDeleted}}
	if err := generator.call(run.adminID, "POST.projects-stories", projects.PostStory, nil, deleteBody, nil); err != nil {
		return err
	}
	err := generator.expect(run, fmt.Sprintf("eliminar H%02d", projectsTypes.StoryNumber(deletedStory.ID)), func(snapshot projectSnapshot) error {
		if _, isFound := findSnapshotStory(snapshot, deletedStory.ID); isFound {
			return fmt.Errorf("la historia eliminada sigue en la lista")
		}
		return nil
	})
	if err != nil || deletedStory.AssigneeID == 0 {
		return err
	}
	return generator.expectAssignment(run, deletedStory.AssigneeID, deletedStory.ID, false)
}

// republishCompanyLogs writes again, with the real clock, the rows of the logs that are not new. The
// history was written with a frozen clock, so its delta-log entries sit in the 10-minute files of
// those past moments, and a browser reads only the files from its cursor on: one that synced the
// company's project list or a person's assigned stories before this run would never see them. The
// project's own logs are new, so their first sync lists the whole view.
func (generator *projectsFlowGenerator) republishCompanyLogs(run *projectFlowRun) error {
	core.SetHistoricalUnix(0)
	if err := generator.call(projectsFlowCreatorID, "POST.projects-projects", projects.PostProject, nil, run.snapshot.project, &run.project); err != nil {
		return err
	}
	assigneeRows := []projectsTypes.StoryAssignee{}
	query := db.Query(&assigneeRows)
	query.CompanyID.Equals(sampleCompanyID).StoryID.In(run.storyIDs...)
	if err := query.Exec(); err != nil || len(assigneeRows) == 0 {
		return err
	}
	assigneeTable := db.TableOf[projectsTypes.StoryAssignee]()
	return db.Update(&assigneeRows, assigneeTable.ProjectID, assigneeTable.AssigneeID, assigneeTable.Status)
}

// checkFinalState reads every story's detail and timeline and every member's assigned stories, and
// checks them against the project: counters, file links, and the per-person index.
func (generator *projectsFlowGenerator) checkFinalState(run *projectFlowRun) error {
	if err := generator.expect(run, "estado final", nil); err != nil {
		return err
	}
	snapshot := run.snapshot
	linkedDocumentIDs := map[int64]bool{}
	for _, story := range snapshot.stories {
		storyNumber := projectsTypes.StoryNumber(story.ID)
		err := generator.checkWithRetries(fmt.Sprintf("estado final de H%02d", storyNumber), func() error {
			detail, err := generator.readStoryDetail(story.ID)
			if err != nil {
				return err
			}
			if len(detail.Attachments) != int(story.AttachmentsCount) {
				return fmt.Errorf("el detalle tiene %d archivos y la historia cuenta %d", len(detail.Attachments), story.AttachmentsCount)
			}
			for _, attachment := range detail.Attachments {
				if err := checkAttachmentDocument(snapshot, attachment, story.ID, 0); err != nil {
					return err
				}
				linkedDocumentIDs[attachment.DocumentID] = true
			}
			activities, err := generator.readStoryActivity(story.ID)
			if err != nil {
				return err
			}
			commentsCount := 0
			for _, activity := range activities {
				if activity.Kind != projectsTypes.ActivityKindComment {
					continue
				}
				commentsCount++
				for _, attachment := range activity.Attachments {
					if err := checkAttachmentDocument(snapshot, attachment, story.ID, activity.ID); err != nil {
						return err
					}
					linkedDocumentIDs[attachment.DocumentID] = true
				}
			}
			if commentsCount != int(story.CommentsCount) {
				return fmt.Errorf("la actividad tiene %d comentarios y la historia cuenta %d", commentsCount, story.CommentsCount)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	for _, document := range snapshot.documents {
		if _, isActiveStory := findSnapshotStory(snapshot, document.StoryID); isActiveStory && !linkedDocumentIDs[document.ID] {
			return fmt.Errorf("estado final: el documento %d (%s) no lo nombra ningún archivo de su historia", document.ID, document.Name)
		}
	}
	run.stats.Files = len(snapshot.documents)

	for _, memberID := range run.memberIDs {
		expectedStoryIDs := []int64{}
		for _, story := range snapshot.stories {
			if story.AssigneeID == memberID {
				expectedStoryIDs = append(expectedStoryIDs, story.ID)
			}
		}
		err := generator.checkWithRetries(fmt.Sprintf("estado final de las historias asignadas a %d", memberID), func() error {
			assigneeRows, err := generator.readAssignedStories(memberID)
			if err != nil {
				return err
			}
			listedStoryIDs := []int64{}
			for _, assigneeRow := range assigneeRows {
				if assigneeRow.ProjectID == run.project.ID {
					listedStoryIDs = append(listedStoryIDs, assigneeRow.StoryID)
				}
			}
			slices.Sort(listedStoryIDs)
			slices.Sort(expectedStoryIDs)
			if !slices.Equal(listedStoryIDs, expectedStoryIDs) {
				return fmt.Errorf("la lista de la persona tiene %v, se esperaba %v", listedStoryIDs, expectedStoryIDs)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// checkProjectInvariants holds for a project after any write: records numbered inside the project,
// each story's final state derived from the sprints that hold it, at most one active sprint, no story
// in two open sprints, finish times exactly on finished cards, and epics, assignees and files of this
// project only.
func checkProjectInvariants(snapshot projectSnapshot) error {
	projectID := snapshot.project.ID
	seenIDs := map[string]bool{}
	checkRecordID := func(kind string, recordID int64, recordProjectID int32) error {
		if recordProjectID != projectID || recordID>>projectsFlowSequenceBits != int64(projectID) || recordID&projectsFlowSequenceMask == 0 {
			return fmt.Errorf("%s %d (proyecto %d) no está numerado dentro del proyecto %d", kind, recordID, recordProjectID, projectID)
		}
		recordKey := fmt.Sprint(kind, recordID)
		if seenIDs[recordKey] {
			return fmt.Errorf("%s %d aparece dos veces en su lista", kind, recordID)
		}
		seenIDs[recordKey] = true
		return nil
	}
	isMember := map[int32]bool{}
	for _, member := range snapshot.project.Members {
		isMember[member.UserID] = true
	}
	activeEpicIDs := map[int64]bool{}
	for _, epic := range snapshot.epics {
		if err := checkRecordID("la épica", epic.ID, epic.ProjectID); err != nil {
			return err
		}
		activeEpicIDs[epic.ID] = true
	}
	storiesByID := map[int64]projectsTypes.Story{}
	for _, story := range snapshot.stories {
		storiesByID[story.ID] = story
	}

	activeSprintName := ""
	openSprintOfStory := map[int64]string{}
	for _, sprint := range snapshot.sprints {
		if err := checkRecordID("el sprint", sprint.ID, sprint.ProjectID); err != nil {
			return err
		}
		isOpen := sprint.State != projectsTypes.SprintStateClosed
		if sprint.State == projectsTypes.SprintStateActive {
			if activeSprintName != "" {
				return fmt.Errorf("hay dos sprints activos: %s y %s", activeSprintName, sprint.Name)
			}
			activeSprintName = sprint.Name
		}
		if !isOpen && (sprint.ClosedAt == 0 || sprint.ClosedBy == 0) {
			return fmt.Errorf("%s está cerrado sin fecha ni autor de cierre", sprint.Name)
		}
		for _, sprintStory := range sprint.Stories {
			storyNumber := projectsTypes.StoryNumber(sprintStory.StoryID)
			if sprintStory.State < projectsTypes.StoryStatePending || sprintStory.State > projectsTypes.StoryStateFinished {
				return fmt.Errorf("H%02d tiene el estado inválido %d en %s", storyNumber, sprintStory.State, sprint.Name)
			}
			if (sprintStory.State == projectsTypes.StoryStateFinished) != (sprintStory.DoneAt > 0) {
				return fmt.Errorf("H%02d en %s: estado %d con fecha de fin %d", storyNumber, sprint.Name, sprintStory.State, sprintStory.DoneAt)
			}
			if !isOpen {
				continue
			}
			if _, isActiveStory := storiesByID[sprintStory.StoryID]; !isActiveStory {
				return fmt.Errorf("%s, abierto, tiene la historia eliminada H%02d", sprint.Name, storyNumber)
			}
			if otherSprintName := openSprintOfStory[sprintStory.StoryID]; otherSprintName != "" {
				return fmt.Errorf("H%02d está en dos sprints abiertos: %s y %s", storyNumber, otherSprintName, sprint.Name)
			}
			if sprintStory.CarriedToSprintID != 0 {
				return fmt.Errorf("H%02d está trasladada en %s, que sigue abierto", storyNumber, sprint.Name)
			}
			openSprintOfStory[sprintStory.StoryID] = sprint.Name
		}
	}

	for _, story := range snapshot.stories {
		storyNumber := projectsTypes.StoryNumber(story.ID)
		if err := checkRecordID("la historia", story.ID, story.ProjectID); err != nil {
			return err
		}
		if story.EpicID != 0 && !activeEpicIDs[story.EpicID] {
			return fmt.Errorf("H%02d está en la épica %d, que no está activa", storyNumber, story.EpicID)
		}
		if story.AssigneeID != 0 && !isMember[story.AssigneeID] {
			return fmt.Errorf("H%02d está asignada a %d, que no es del equipo", storyNumber, story.AssigneeID)
		}
		// The final state is the state in the sprint the story was most recently added to (ties: the
		// highest sprint ID), recomputed here from the sprints as stored.
		expectedState, expectedLastSprintID, expectedSprintIDs := projectsTypes.StoryStateNotPlanned, int64(0), []int64{}
		latestAddedAt := int32(-1)
		for _, sprint := range snapshot.sprints {
			for _, sprintStory := range sprint.Stories {
				if sprintStory.StoryID != story.ID {
					continue
				}
				expectedSprintIDs = append(expectedSprintIDs, sprint.ID)
				if sprintStory.AddedAt > latestAddedAt || (sprintStory.AddedAt == latestAddedAt && sprint.ID > expectedLastSprintID) {
					latestAddedAt, expectedLastSprintID, expectedState = sprintStory.AddedAt, sprint.ID, sprintStory.State
				}
			}
		}
		slices.Sort(expectedSprintIDs)
		if story.State != expectedState || story.LastSprintID != expectedLastSprintID || !slices.Equal(story.SprintIDs, expectedSprintIDs) {
			return fmt.Errorf("H%02d: estado %d, último sprint %d, sprints %v; según los sprints debería ser %d, %d, %v",
				storyNumber, story.State, story.LastSprintID, story.SprintIDs, expectedState, expectedLastSprintID, expectedSprintIDs)
		}
		if story.State != projectsTypes.StoryStateNotPlanned &&
			!slices.ContainsFunc(story.StateDates, func(stateDate projectsTypes.StoryStateDate) bool { return stateDate.State == story.State }) {
			return fmt.Errorf("H%02d está en el estado %d sin su fecha en StateDates: %v", storyNumber, story.State, story.StateDates)
		}
	}

	projectFolder := fmt.Sprintf("projects/c%d/p%d/", sampleCompanyID, projectID)
	for _, document := range snapshot.documents {
		if err := checkRecordID("el documento", document.ID, document.ProjectID); err != nil {
			return err
		}
		if !strings.HasPrefix(document.Key, projectFolder) {
			return fmt.Errorf("el documento %d apunta fuera de la carpeta del proyecto: %s", document.ID, document.Key)
		}
	}
	return nil
}

// checkAttachmentDocument: an attached file is listed on the Documents page, by its DocumentID, with
// the same file, story and comment.
func checkAttachmentDocument(snapshot projectSnapshot, attachment projectsTypes.Attachment, storyID int64, activityID int64) error {
	documentIndex := slices.IndexFunc(snapshot.documents, func(document projectsTypes.Document) bool { return document.ID == attachment.DocumentID })
	if attachment.DocumentID == 0 || documentIndex < 0 {
		return fmt.Errorf("el archivo %s no está en Documentos (DocumentID %d)", attachment.Name, attachment.DocumentID)
	}
	document := snapshot.documents[documentIndex]
	if document.Key != attachment.Key || document.StoryID != storyID || document.ActivityID != activityID {
		return fmt.Errorf("el documento %d no coincide con su archivo %s: historia %d, comentario %d", document.ID, attachment.Name, document.StoryID, document.ActivityID)
	}
	return nil
}

// expect reads the project back through its GET routes and checks every invariant of a project plus
// the step's own expectation (stepCheck, optional). The snapshot read is kept for the next step.
func (generator *projectsFlowGenerator) expect(run *projectFlowRun, step string, stepCheck func(snapshot projectSnapshot) error) error {
	return generator.checkWithRetries(step, func() error {
		snapshot, err := generator.loadSnapshot(run.project.ID)
		if err != nil {
			return err
		}
		if err := checkProjectInvariants(snapshot); err != nil {
			return err
		}
		if stepCheck != nil {
			if err := stepCheck(snapshot); err != nil {
				return err
			}
		}
		run.snapshot = snapshot
		return nil
	})
}

// checkWithRetries runs a check, reading again a few times before it fails: the lists come from
// materialized views, which may trail the write just acknowledged. A check that only passes on a
// later read is counted and logged.
func (generator *projectsFlowGenerator) checkWithRetries(step string, check func() error) error {
	var checkErr error
	for attempt := 1; attempt <= projectsFlowCheckAttempts; attempt++ {
		if checkErr = check(); checkErr == nil {
			if attempt > 1 {
				generator.viewRetries++
				core.Log("ProjectsFlow:: lectura desfasada, se leyó", attempt, "veces:", step)
			}
			generator.checksPassed++
			return nil
		}
		time.Sleep(time.Duration(attempt) * 300 * time.Millisecond)
	}
	return fmt.Errorf("%s: %w", step, checkErr)
}

// expectAssignment checks a person's assigned stories (the by-person report's list): the story is in
// it, under its project, or it is not.
func (generator *projectsFlowGenerator) expectAssignment(run *projectFlowRun, personID int32, storyID int64, isAssigned bool) error {
	step := fmt.Sprintf("H%02d en la lista de %d: %v", projectsTypes.StoryNumber(storyID), personID, isAssigned)
	return generator.checkWithRetries(step, func() error {
		assigneeRows, err := generator.readAssignedStories(personID)
		if err != nil {
			return err
		}
		rowIndex := slices.IndexFunc(assigneeRows, func(assigneeRow projectsTypes.StoryAssignee) bool { return assigneeRow.StoryID == storyID })
		if (rowIndex >= 0) != isAssigned || (isAssigned && assigneeRows[rowIndex].ProjectID != run.project.ID) {
			return fmt.Errorf("la lista de la persona no coincide (fila %d)", rowIndex)
		}
		return nil
	})
}

// loadSnapshot reads the project the way the browser does on a first sync of each route.
func (generator *projectsFlowGenerator) loadSnapshot(projectID int32) (projectSnapshot, error) {
	snapshot := projectSnapshot{}
	companyProjects, err := readDeltaList[projectsTypes.Project](generator, "GET.projects-projects", projects.GetProjects, nil)
	if err != nil {
		return snapshot, err
	}
	projectIndex := slices.IndexFunc(companyProjects, func(project projectsTypes.Project) bool { return project.ID == projectID })
	if projectIndex < 0 {
		return snapshot, fmt.Errorf("el proyecto %d no está en GET.projects-projects", projectID)
	}
	snapshot.project = companyProjects[projectIndex]
	projectQuery := map[string]string{"project-id": strconv.Itoa(int(projectID))}
	if snapshot.epics, err = readDeltaList[projectsTypes.Epic](generator, "GET.projects-epics", projects.GetEpics, projectQuery); err != nil {
		return snapshot, err
	}
	if snapshot.stories, err = readDeltaList[projectsTypes.Story](generator, "GET.projects-stories", projects.GetStories, projectQuery); err != nil {
		return snapshot, err
	}
	if snapshot.sprints, err = readDeltaList[projectsTypes.Sprint](generator, "GET.projects-sprints", projects.GetSprints, projectQuery); err != nil {
		return snapshot, err
	}
	snapshot.documents, err = readDeltaList[projectsTypes.Document](generator, "GET.projects-documents", projects.GetDocuments, projectQuery)
	return snapshot, err
}

func (generator *projectsFlowGenerator) readStoryDetail(storyID int64) (projectsTypes.StoryDetail, error) {
	details, err := readDeltaList[projectsTypes.StoryDetail](generator, "GET.projects-story-detail", projects.GetStoryDetail,
		map[string]string{"story-id": strconv.FormatInt(storyID, 10)})
	if err != nil {
		return projectsTypes.StoryDetail{}, err
	}
	if len(details) != 1 {
		return projectsTypes.StoryDetail{}, fmt.Errorf("la historia %d tiene %d detalles", storyID, len(details))
	}
	return details[0], nil
}

func (generator *projectsFlowGenerator) readStoryActivity(storyID int64) ([]projectsTypes.StoryActivity, error) {
	return readDeltaList[projectsTypes.StoryActivity](generator, "GET.projects-story-activity", projects.GetStoryActivity,
		map[string]string{"story-id": strconv.FormatInt(storyID, 10)})
}

func (generator *projectsFlowGenerator) readAssignedStories(personID int32) ([]projectsTypes.StoryAssignee, error) {
	return readDeltaList[projectsTypes.StoryAssignee](generator, "GET.projects-assigned-stories", projects.GetAssignedStories,
		map[string]string{"user-id": strconv.Itoa(int(personID))})
}

// readDeltaList reads a delta route as a first sync does (no cursor): the whole active list. The
// response's Metadata records (one per log file, or the replace marker) are not records.
func readDeltaList[T any](generator *projectsFlowGenerator, route string, handler projectsFlowHandler, query map[string]string) ([]T, error) {
	rawRecords := []json.RawMessage{}
	if err := generator.call(projectsFlowCreatorID, route, handler, query, nil, &rawRecords); err != nil {
		return nil, err
	}
	records := make([]T, 0, len(rawRecords))
	for _, rawRecord := range rawRecords {
		metadataProbe := struct {
			Metadata *string `json:"mdt"`
		}{}
		if err := json.Unmarshal(rawRecord, &metadataProbe); err != nil {
			return nil, fmt.Errorf("%s: %w", route, err)
		}
		if metadataProbe.Metadata != nil {
			continue
		}
		var record T
		if err := json.Unmarshal(rawRecord, &record); err != nil {
			return nil, fmt.Errorf("%s: %w", route, err)
		}
		records = append(records, record)
	}
	return records, nil
}

// editStory saves a story from its form: the stored story and detail, changed by edit.
func (generator *projectsFlowGenerator) editStory(run *projectFlowRun, actorID int32, storyID int64, edit func(story *projectsTypes.Story, detail *projectsTypes.StoryDetail)) (projectsFlowStoryWithDetail, error) {
	story, isFound := findSnapshotStory(run.snapshot, storyID)
	if !isFound {
		return projectsFlowStoryWithDetail{}, fmt.Errorf("la historia %d no está en el proyecto", storyID)
	}
	detail, err := generator.readStoryDetail(storyID)
	if err != nil {
		return projectsFlowStoryWithDetail{}, err
	}
	edit(&story, &detail)
	generator.tick()
	savedStory := projectsFlowStoryWithDetail{}
	err = generator.call(actorID, "POST.projects-stories", projects.PostStory, nil, projectsFlowStoryWithDetail{Story: story, Detail: detail}, &savedStory)
	return savedStory, err
}

func (generator *projectsFlowGenerator) uploadFile(run *projectFlowRun, actorID int32, name string, contentType string, content string) (projectsTypes.Attachment, error) {
	generator.tick()
	uploaded := projectsTypes.Attachment{}
	uploadBody := map[string]any{
		"ProjectID": run.project.ID, "Name": name, "ContentType": contentType, "Content": base64.StdEncoding.EncodeToString([]byte(content)),
	}
	if err := generator.call(actorID, "POST.projects-attachment-upload", projects.PostAttachmentUpload, nil, uploadBody, &uploaded); err != nil {
		return uploaded, err
	}
	if uploaded.Key == "" {
		return uploaded, fmt.Errorf("la subida de %s no devolvió la ruta del archivo", name)
	}
	return uploaded, nil
}

// call runs a production handler as the given user, as the router would after authenticating it.
func (generator *projectsFlowGenerator) call(actorID int32, route string, handler projectsFlowHandler, query map[string]string, payload any, target any) error {
	body := ""
	if payload != nil {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = string(bodyBytes)
	}
	method, _, _ := strings.Cut(route, ".")
	request := core.HandlerArgs{
		Body: &body, Query: query, Route: route, Method: method,
		User: &core.UsuarioToken{CompanyID: sampleCompanyID, ID: actorID},
	}
	response := handler(&request)
	if response.StatusCode != 200 {
		return fmt.Errorf("%s (usuario %d) respondió %d: %s", route, actorID, response.StatusCode, response.Error)
	}
	if target == nil {
		return nil
	}
	if response.Body == nil {
		return fmt.Errorf("%s respondió vacío", route)
	}
	return json.Unmarshal(*response.Body, target)
}

// startDay moves the simulated clock to 08:00 of day.
func (generator *projectsFlowGenerator) startDay(day int16) {
	generator.clockDay, generator.clockMinute = day, 8*60
}

// tick freezes the clock a few minutes after the previous write, so the writes of a day land in order
// and each with its own SUnixTime (2-second resolution).
func (generator *projectsFlowGenerator) tick() {
	generator.clockMinute += 2 + generator.random.Intn(6)
	core.SetHistoricalUnix(generator.dayStart(generator.clockDay).Add(time.Duration(generator.clockMinute) * time.Minute).Unix())
}

// dayStart is the local midnight of a UnixDay, the day core.FechaUnix() reports for it.
func (generator *projectsFlowGenerator) dayStart(day int16) time.Time {
	dayInTarget := generator.realNow.AddDate(0, 0, int(day-generator.today))
	return time.Date(dayInTarget.Year(), dayInTarget.Month(), dayInTarget.Day(), 0, 0, 0, 0, dayInTarget.Location())
}

func (generator *projectsFlowGenerator) pickMember(run *projectFlowRun) int32 {
	return run.memberIDs[generator.random.Intn(len(run.memberIDs))]
}

// storyWorker is who moves a story's card: its assignee, or the admin when it has none.
func (generator *projectsFlowGenerator) storyWorker(run *projectFlowRun, storyID int64) int32 {
	if story, _ := findSnapshotStory(run.snapshot, storyID); story.AssigneeID != 0 {
		return story.AssigneeID
	}
	return run.adminID
}

func findSnapshotStory(snapshot projectSnapshot, storyID int64) (projectsTypes.Story, bool) {
	storyIndex := slices.IndexFunc(snapshot.stories, func(story projectsTypes.Story) bool { return story.ID == storyID })
	if storyIndex < 0 {
		return projectsTypes.Story{}, false
	}
	return snapshot.stories[storyIndex], true
}

func findSnapshotSprint(snapshot projectSnapshot, sprintID int64) (projectsTypes.Sprint, bool) {
	sprintIndex := slices.IndexFunc(snapshot.sprints, func(sprint projectsTypes.Sprint) bool { return sprint.ID == sprintID })
	if sprintIndex < 0 {
		return projectsTypes.Sprint{}, false
	}
	return snapshot.sprints[sprintIndex], true
}

func sprintStoryPosition(sprint projectsTypes.Sprint, storyID int64) int {
	return slices.IndexFunc(sprint.Stories, func(sprintStory projectsTypes.SprintStory) bool { return sprintStory.StoryID == storyID })
}

func sameKeys(leftKeys []string, rightKeys []string) bool {
	leftKeys, rightKeys = slices.Clone(leftKeys), slices.Clone(rightKeys)
	slices.Sort(leftKeys)
	slices.Sort(rightKeys)
	return slices.Equal(leftKeys, rightKeys)
}

func compareFloat(left float64, right float64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	}
	return 0
}

func projectsFlowDate(unixDay int16) string {
	return time.Unix(int64(unixDay)*86400, 0).UTC().Format("02/01/2006")
}

// The generated files: small, readable, and in the project's language.

func projectsFlowSpecificationText(storyTitle string) string {
	return "Especificación funcional\n\nHistoria: " + storyTitle + "\n\n" +
		"1. Alcance: el flujo descrito en la historia, de punta a punta.\n" +
		"2. Validaciones: cada campo obligatorio muestra su mensaje de error.\n" +
		"3. Permisos: solo los usuarios con acceso pueden ejecutar la acción.\n"
}

func projectsFlowMockupSVG(storyTitle string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="480" height="270" viewBox="0 0 480 270">` +
		`<rect width="480" height="270" fill="#f4f4f8"/><rect x="20" y="20" width="440" height="40" rx="6" fill="#4f46e5"/>` +
		`<text x="36" y="46" font-family="sans-serif" font-size="16" fill="#fff">` + html.EscapeString(storyTitle) + `</text>` +
		`<rect x="20" y="80" width="440" height="170" rx="6" fill="#fff" stroke="#d4d4dc"/></svg>`
}

func projectsFlowTestCasesCSV(storyTitle string) string {
	return "Caso,Resultado esperado,Estado\n" +
		"\"" + storyTitle + ": flujo principal\",Se completa sin errores,Aprobado\n" +
		"\"" + storyTitle + ": datos inválidos\",Muestra el mensaje de error,Aprobado\n"
}

func projectsFlowCharterText(project projectsTypes.Project) string {
	return "Acta de constitución del proyecto\n\nProyecto: " + project.Name + "\n\n" + project.Description + "\n\n" +
		"Inicio: " + projectsFlowDate(project.StartDate) + "\nFin previsto: " + projectsFlowDate(project.EndDate) + "\n"
}
