package projects

import "app/core"

// ModuleHandlers are the routes of the agile projects module. Route names carry the module prefix
// ("projects-"), and every write route is mapped in backend/access.toml.
var ModuleHandlers = core.AppRouterType{
	"GET.projects-projects":           GetProjects,
	"POST.projects-projects":          PostProject,
	"GET.projects-epics":              GetEpics,
	"POST.projects-epics":             PostEpics,
	"GET.projects-stories":            GetStories,
	"GET.projects-stories-ids":        GetStoriesByIDs,
	"GET.projects-assigned-stories":   GetAssignedStories,
	"POST.projects-stories":           PostStory,
	"POST.projects-stories-rank":      PostStoryRank,
	"GET.projects-story-details-ids":  GetStoryDetailsByIDs,
	"GET.projects-story-activity":     GetStoryActivity,
	"POST.projects-story-comments":    PostStoryComment,
	"POST.projects-attachment-upload": PostAttachmentUpload,
	"GET.projects-sprints":            GetSprints,
	"POST.projects-sprints":           PostSprint,
	"POST.projects-sprint-start":      PostSprintStart,
	"POST.projects-sprint-move":       PostSprintMove,
	"POST.projects-sprint-close":      PostSprintClose,
	"GET.projects-documents":          GetDocuments,
	"POST.projects-documents":         PostDocument,
}
