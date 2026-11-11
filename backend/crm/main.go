package crm

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"GET.client-provider":              GetClientProviders,
	"GET.client-provider-ids":          GetClientProvidersByIDs,
	"GET.client-provider-snapshot-ids": GetClientProviderSnapshotsByIDs,
	"POST.client-provider":             PostClientProviders,

	"GET.crm-pipelines":               GetCrmPipelines,
	"POST.crm-pipeline":               PostCrmPipeline,
	"POST.crm-pipeline-members":       PostCrmPipelineMembers,
	"GET.crm-custom-fields":           GetCrmCustomFields,
	"POST.crm-custom-field":           PostCrmCustomField,
	"GET.crm-lost-reasons":            GetCrmLostReasons,
	"POST.crm-lost-reason":            PostCrmLostReason,
	"GET.crm-email-templates":         GetCrmEmailTemplates,
	"POST.crm-email-template":         PostCrmEmailTemplate,
	"GET.crm-opportunity-email-draft": GetCrmOpportunityEmailDraft,
	"POST.crm-opportunity-email":      PostCrmOpportunityEmail,
	"GET.crm-activity-types":          GetCrmActivityTypes,
	"POST.crm-activity-type":          PostCrmActivityType,

	"GET.crm-opportunities":         GetCrmOpportunities,
	"GET.crm-opportunity-log":       GetCrmOpportunityLog,
	"POST.crm-opportunity":          PostCrmOpportunity,
	"POST.crm-opportunity-move":     PostCrmOpportunityMove,
	"POST.crm-opportunity-close":    PostCrmOpportunityClose,
	"GET.crm-opportunity-activity":  GetCrmOpportunityActivity,
	"POST.crm-opportunity-activity": PostCrmOpportunityActivity,
	"POST.crm-attachment-upload":    PostCrmAttachmentUpload,

	"GET.crm-teams":                GetCrmTeams,
	"POST.crm-team":                PostCrmTeam,
	"GET.crm-goals":                GetCrmGoals,
	"POST.crm-goals":               PostCrmGoals,
	"GET.crm-forecast-submissions": GetCrmForecastSubmissions,
	"POST.crm-forecast-submission": PostCrmForecastSubmission,

	"GET.crm-tasks":                GetCrmTasks,
	"POST.crm-task":                PostCrmTask,
	"POST.crm-task-decide":         PostCrmTaskDecide,
	"GET.crm-opportunity-approval": GetCrmOpportunityApproval,

	"GET.crm-pipeline-opportunity-status": GetCrmPipelineOpportunityStatus,
	"GET.crm-opportunity-ids":             GetCrmOpportunitiesByIDs,

	"GET.crm-report-scope":              GetCrmReportScope,
	"GET.crm-report-stage-log":          GetCrmReportStageLog,
	"GET.crm-report-open-opportunities": GetCrmReportOpenOpportunities,
	"GET.crm-report-activities":         GetCrmReportActivities,
}
