package routing

type ResponseCategory string

const (
	ResponseSocial                    ResponseCategory = "social"
	ResponseOutOfScope                ResponseCategory = "out_of_scope"
	ResponseOperationalUnavailable    ResponseCategory = "operational_unavailable"
	ResponseClassificationUnavailable ResponseCategory = "classification_unavailable"
	ResponseBuilderStateChanged       ResponseCategory = "builder_state_changed"
	ResponseDocumentationMissing      ResponseCategory = "documentation_missing"
	ResponseDocumentationUnavailable  ResponseCategory = "documentation_unavailable"
	ResponseTurnFailed                ResponseCategory = "turn_failed"
)

var localizedResponses = map[ResponseCategory]map[Language]string{
	ResponseSocial: {
		LanguageSpanish: "Hola. Soy el asistente de Genix. Puedo ayudarte a entender el sistema, encontrar funciones y trabajar en la página actual.",
		LanguageEnglish: "Hello. I’m the Genix assistant. I can help you understand the system, find features, and work with the current page.",
	},
	ResponseOutOfScope: {
		LanguageSpanish: "Soy el asistente de Genix y solo puedo ayudarte con el sistema, sus procesos y la información disponible dentro de Genix.",
		LanguageEnglish: "I’m the Genix assistant and can only help with the system, its workflows, and information available through Genix.",
	},
	ResponseOperationalUnavailable: {
		LanguageSpanish: "Esa consulta de datos de la empresa todavía no está disponible. No se consultó ningún registro.",
		LanguageEnglish: "That company-data query is not available yet. No records were searched.",
	},
	ResponseClassificationUnavailable: {
		LanguageSpanish: "No pude interpretar la solicitud en este momento. Inténtalo nuevamente en unos segundos.",
		LanguageEnglish: "I couldn’t interpret the request right now. Please try again in a few seconds.",
	},
	ResponseBuilderStateChanged: {
		LanguageSpanish: "La página o sección seleccionada cambió durante la solicitud. Vuelve a seleccionarla e inténtalo otra vez.",
		LanguageEnglish: "The selected page or section changed during the request. Select it again and retry.",
	},
	ResponseDocumentationMissing: {
		LanguageSpanish: "La documentación actual de Genix no establece esa respuesta.",
		LanguageEnglish: "The current Genix documentation does not establish that answer.",
	},
	ResponseDocumentationUnavailable: {
		LanguageSpanish: "No pude consultar la documentación de Genix en este momento. Inténtalo nuevamente en unos segundos.",
		LanguageEnglish: "I couldn’t access the Genix documentation right now. Please try again in a few seconds.",
	},
	ResponseTurnFailed: {
		LanguageSpanish: "No pude completar la solicitud en este momento. No se aplicó ninguna acción no confirmada; inténtalo nuevamente.",
		LanguageEnglish: "I couldn’t complete the request right now. No unconfirmed action was applied; please try again.",
	},
}

func LocalizedResponse(category ResponseCategory, language Language) string {
	if language != LanguageEnglish {
		language = LanguageSpanish
	}
	return localizedResponses[category][language]
}
