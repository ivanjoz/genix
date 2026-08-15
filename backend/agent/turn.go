package agent

// One user turn is one plain POST that blocks until the turn finishes. Its
// response carries no content beyond `{Ok:true}`: everything the turn produces —
// chat events (agentStatus / agentReply / agentError / agentSections) and page
// commands (navigate / agent.invoke / getPageContent / getMenu) — travels down
// the tab's already-open stream, which the browser opened before sending this.
//
// This shape is what makes the Lambda deployment possible: a Function URL
// invocation cannot hold a stream for a whole turn, and the browser's reply to a
// command cannot arrive inside the same invocation. The stream and the replies
// live on the SSE bridge instead (bridge.go), and the only thing the turn needs
// from HTTP is a request that stays open while the loop runs.

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"app/agent/routing"
	"app/core"
)

// turnTimeout caps a single turn end to end: long enough for a dozen LLM
// round-trips, short enough that a wedged turn cannot hold the session's
// inFlight guard forever.
//
// It must stay *below* the Lambda's own Timeout (300s in cloud/template.yml).
// If they were equal the invocation would be killed at the same instant the
// context expires, leaving no room to push the agentError/turnEnd events and
// wedging the widget's busy state.
const turnTimeout = 4 * time.Minute

// TurnRequest is the `POST p-agent-turn` body. Message/ModelHash/Timestamp/ModeID,
// compact Surface, and AppLanguage mirror ChatUserMessage. Channel names the
// stream this turn must talk to (channel.go); SessionID scopes the persisted
// history and is minted by the client so it survives a Lambda instance change
// mid-conversation. Path is the SPA route the user is on, which enriches
// progress labels.
type TurnRequest struct {
	Channel     string
	SessionID   int64
	Message     string
	ModelHash   string
	Timestamp   int64
	ModeID      int
	Path        string
	Surface     routing.SurfaceContext
	AppLanguage routing.Language
}

type TurnResponse struct {
	Ok bool
}

// PostAgentTurn runs one chat turn and returns when it is done.
//
// The route is registered as `p-agent-turn`: in mainHandler the `p-` prefix only
// means "no acceso requirement", which is what the agent chat needs (any signed
// in user may use it), so the session token is validated right here instead. A
// POST without an access_list entry would otherwise be rejected for every user
// except the admin.
func PostAgentTurn(req *core.HandlerArgs) core.HandlerResponse {
	req.User = core.CheckUser(req, 0)
	if len(req.User.Error) > 0 {
		return req.MakeErr401(req.User.Error)
	}

	turnRequest := TurnRequest{}
	if req.Body == nil || len(*req.Body) == 0 {
		return req.MakeErr("No se recibió el cuerpo de la petición del turno.")
	}
	if err := json.Unmarshal([]byte(*req.Body), &turnRequest); err != nil {
		return req.MakeErr("El cuerpo de la petición del turno no es un JSON válido:", err)
	}

	channelToken := strings.TrimSpace(turnRequest.Channel)
	channelCompanyID, channelUserID, tab, channelError := DecodeChannelToken(channelToken)
	if channelError != nil {
		return req.MakeErr("Channel inválido:", channelError)
	}
	// The channel token names the stream but proves nothing — it is the session
	// token that identifies the caller. If they disagree, the client is trying to
	// drive another tenant's tab.
	if channelCompanyID != req.User.CompanyID || channelUserID != req.User.ID {
		core.Log("agent.turn channel ajeno tab::", shortTabID(tab), " channel::", channelCompanyID, "/", channelUserID,
			" token::", req.User.CompanyID, "/", req.User.ID)
		return req.MakeErr401("El canal solicitado no pertenece al usuario autenticado.")
	}
	if len(strings.TrimSpace(turnRequest.Message)) == 0 {
		return req.MakeErr("El mensaje del turno está vacío.")
	}

	session := ensureChatSession(tab, channelToken, req.User.CompanyID, req.User.ID, turnRequest.SessionID, turnRequest.Path)
	// One turn per tab. Note this only guards within a single process: under
	// Lambda two concurrent turns could land on different execution environments.
	if !session.inFlight.CompareAndSwap(false, true) {
		core.Log("agent.turn busy tab::", shortTabID(tab), " incoming_bytes::", len(turnRequest.Message))
		return req.MakeErr("Ya hay un turno en ejecución para esta pestaña.")
	}
	defer session.inFlight.Store(false)

	// Outside Lambda the request context cancels the turn when the browser goes
	// away; in Lambda there is no *http.Request, so the timeout is the only bound.
	parentContext := context.Background()
	if req.ReqContext != nil {
		parentContext = req.ReqContext.Context()
	}
	runContext, cancelRun := context.WithTimeout(parentContext, turnTimeout)
	defer cancelRun()
	if rateLimitError := core.ChargeAPIUsage(
		runContext, req.User.CompanyID, req.User.ID, req.RouteID, "POST", len(*req.Body),
	); rateLimitError != nil {
		core.Log("agent.turn base credit rejected tab::", shortTabID(tab), " err::", rateLimitError)
		return req.MakeCreditRateLimitResponse(rateLimitError)
	}
	// Every nested LLM request inherits the authenticated identity and the route that opened the
	// turn, so what a turn spends on inference lands on POST.p-agent-turn rather than in a bucket
	// of its own.
	runContext = core.WithCreditRateLimitIdentity(
		runContext, req.User.CompanyID, req.User.ID, req.RouteID,
	)

	core.Log("agent.turn start tab::", shortTabID(tab), " company::", session.CompanyID, " user::", session.UserID,
		" session::", session.SessionID, " mode::", turnRequest.ModeID, " path::", turnRequest.Path, " bridged::", BridgeEnabled())

	if err := session.RunUserMessage(runContext, ChatUserMessage{
		Message:     turnRequest.Message,
		ModelHash:   turnRequest.ModelHash,
		Timestamp:   turnRequest.Timestamp,
		ModeID:      turnRequest.ModeID,
		Surface:     turnRequest.Surface,
		AppLanguage: turnRequest.AppLanguage,
	}); err != nil {
		core.Log("agent.turn RunUserMessage error tab::", shortTabID(tab), " err::", err)
		if core.IsCreditRateLimitError(err) {
			return req.MakeCreditRateLimitResponse(err)
		}
		// Reported down the stream, not as an HTTP error: the widget renders agent
		// failures inline. Errors reaching this point happened before a safe
		// final reply could be persisted, so avoid leaking provider internals.
		session.sendError("No se pudo iniciar el turno del asistente.")
	}

	// turnEnd closes the widget's busy state. The stream itself stays open for
	// the next turn.
	if err := session.pushEvent([]byte(`{"Type":"` + ChatTypeTurnEnd + `"}`)); err != nil {
		core.Log("agent.turn no se pudo enviar turnEnd tab::", shortTabID(tab), " err::", err)
	}
	core.Log("agent.turn end tab::", shortTabID(tab))

	return req.MakeResponse(TurnResponse{Ok: true})
}
