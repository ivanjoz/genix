package handlers

import (
	"app/core"
	servermetrics "app/system"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func GetSystemMetricsStream(req *core.HandlerArgs) core.HandlerResponse {
	if !core.Env.IS_LOCAL {
		return req.MakeErr("La API SSE de métricas solo está disponible en modo servidor local/VPS.")
	}

	if req.Usuario == nil || req.Usuario.ID == 0 {
		return req.MakeErr401("No autorizado para consumir métricas del servidor.")
	}

	intervalMilliseconds, intervalError := parseIntervalMilliseconds(req.GetQuery("interval_ms"))
	if intervalError != nil {
		return req.MakeErr("interval_ms inválido. Debe estar entre 500 y 10000 ms.")
	}

	mountPath := strings.TrimSpace(req.GetQuery("mount"))
	if len(mountPath) == 0 {
		mountPath = "/"
	}

	networkInterfaceName := strings.TrimSpace(req.GetQuery("iface"))
	metricsCollector := servermetrics.NewServerMetricsCollector(networkInterfaceName)

	streamStartedPayload := map[string]any{
		"timestamp_unix": time.Now().Unix(),
		"interval_ms":    intervalMilliseconds,
		"mount_path":     mountPath,
		"interface_name": networkInterfaceName,
	}

	if sendError := core.SendServerEvent(req, "connected", streamStartedPayload); sendError != nil {
		return req.MakeErr("No se pudo iniciar el stream SSE de métricas:", sendError)
	}

	core.Log("SSE metrics stream iniciado | user:", req.Usuario.ID, "| route:", req.Route)

	streamTicker := time.NewTicker(time.Duration(intervalMilliseconds) * time.Millisecond)
	defer streamTicker.Stop()

	lastKeepAliveUnix := time.Now().Unix()

	for {
		select {
		case <-req.ReqContext.Context().Done():
			core.Log("SSE metrics stream finalizado por disconnect | user:", req.Usuario.ID, "| route:", req.Route)
			return core.HandlerResponse{StatusCode: http.StatusOK, StreamHandled: true}
		case <-streamTicker.C:
			activeHTTPConnections := core.GetActiveHTTPConnections()
			metricsSnapshot, metricsWarnings := metricsCollector.CollectSnapshot(mountPath, activeHTTPConnections)

			metricsEventName := "metrics"
			if len(metricsWarnings) > 0 {
				metricsEventName = "warning"
			}

			metricsPayload := map[string]any{
				"metrics":  metricsSnapshot,
				"warnings": metricsWarnings,
			}

			if sendError := core.SendServerEvent(req, metricsEventName, metricsPayload); sendError != nil {
				core.Log("SSE metrics stream write error | user:", req.Usuario.ID, "| error:", sendError.Error())
				return core.HandlerResponse{StatusCode: http.StatusOK, StreamHandled: true}
			}

			// Send lightweight keepalive comments to reduce idle proxy interruptions.
			nowUnix := time.Now().Unix()
			if nowUnix-lastKeepAliveUnix >= 15 {
				if sendCommentError := core.SendServerComment(req, "metrics-stream-alive"); sendCommentError != nil {
					core.Log("SSE keepalive comment failed | user:", req.Usuario.ID, "| error:", sendCommentError.Error())
					return core.HandlerResponse{StatusCode: http.StatusOK, StreamHandled: true}
				}
				lastKeepAliveUnix = nowUnix
			}
		}
	}
}

func parseIntervalMilliseconds(intervalRawValue string) (int64, error) {
	cleanIntervalValue := strings.TrimSpace(intervalRawValue)
	if len(cleanIntervalValue) == 0 {
		return 1000, nil
	}

	parsedIntervalValue, parseError := strconv.ParseInt(cleanIntervalValue, 10, 64)
	if parseError != nil {
		return 0, parseError
	}
	if parsedIntervalValue < 500 || parsedIntervalValue > 10000 {
		return 0, errors.New("interval fuera de rango")
	}
	return parsedIntervalValue, nil
}
