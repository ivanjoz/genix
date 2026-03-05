package handlers

import (
	"app/core"
	servermetrics "app/system"
	"strconv"
	"strings"
)

func GetSystemMemoryPackages(req *core.HandlerArgs) core.HandlerResponse {
	if !core.Env.IS_LOCAL {
		return req.MakeErr("La API de memoria por paquetes solo está disponible en modo servidor local/VPS.")
	}

	if req.Usuario == nil || req.Usuario.ID == 0 {
		return req.MakeErr401("No autorizado para consultar memoria del servidor.")
	}

	requestedLimit, parseLimitError := parsePackageLimit(req.GetQuery("limit"))
	if parseLimitError != nil {
		return req.MakeErr("limit inválido. Debe estar entre 1 y 100.")
	}

	core.Log("System memory packages report solicitado | user:", req.Usuario.ID, "| route:", req.Route, "| limit:", requestedLimit)

	goHeapPackageReport, reportWarnings := servermetrics.CollectGoHeapPackageReport(requestedLimit)
	responsePayload := map[string]any{
		"report":   goHeapPackageReport,
		"warnings": reportWarnings,
	}

	core.Log("System memory packages report generado | user:", req.Usuario.ID, "| packages:", len(goHeapPackageReport.TopPackages), "| warnings:", len(reportWarnings), "| go_heap_inuse:", goHeapPackageReport.GoHeapInuseBytes)

	return req.MakeResponse(responsePayload)
}

func parsePackageLimit(rawPackageLimitValue string) (int, error) {
	cleanPackageLimitValue := strings.TrimSpace(rawPackageLimitValue)
	if len(cleanPackageLimitValue) == 0 {
		return 20, nil
	}

	parsedPackageLimitValue, parseLimitError := strconv.Atoi(cleanPackageLimitValue)
	if parseLimitError != nil {
		return 0, parseLimitError
	}
	if parsedPackageLimitValue < 1 || parsedPackageLimitValue > 100 {
		return 0, strconv.ErrRange
	}
	return parsedPackageLimitValue, nil
}
