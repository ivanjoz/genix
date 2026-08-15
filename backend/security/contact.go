package security

import (
	"app/core"
	"app/db"
	"app/security/types"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"unicode/utf8"
)

// Public contact form: /welcome posts a name, an address and a message, and we store the message
// and mail it to the inbox configured in [contact]. The endpoint is public ("p-" prefix), which in
// this backend also means it bypasses the credit rate limiter (main-handlers.go), so the only
// brake is the per-IP window enforced below — the same shape as public sign-up, under its own lock
// action so the two forms never queue behind each other.

const (
	contactStatusDelivered   = int8(1)
	contactStatusUndelivered = int8(2)

	// Field ceilings, counted in runes so an accented Spanish message is not cut short of what the
	// form's own limit implies. The row is built from an unauthenticated body, so the sizes are
	// enforced here rather than trusted from the browser.
	contactMaxNameRunes    = 120
	contactMaxEmailRunes   = 160
	contactMaxCompanyRunes = 160
	contactMaxMessageRunes = 4000
	// Below this a submission carries nothing to answer, and it is the length a bot filling every
	// field with one character trips over.
	contactMinMessageRunes = 10
	contactMinNameRunes    = 2
)

const contactEmailSubject = "Genix — nuevo mensaje de contacto"

// countRecentMessagesFromIP is how many messages this client has sent inside the window. Sign-up
// counts distinct addresses because retrying one address there is legitimate; here every
// submission is a fresh message and a fresh email, so every one of them counts.
//
// The query is a clustering slice — (WeekCode =, IP =, Created BETWEEN) is a key prefix with a
// range on the next column — so it reads only the rows inside the window, not every message the
// address ever sent this week.
//
// Correct only while the caller holds the per-IP lock: without it, parallel Lambdas all read the
// same count, all conclude they are under the limit, and all insert.
func countRecentMessagesFromIP(ipKey int64) (int32, error) {
	nowTime := core.SUnixTime()
	windowStart := nowTime - (core.Env.CONTACT_WINDOW_MINUTES * 30) // 1 SUnixTime unit = 2s
	total := int32(0)

	for _, weekCode := range recentWeekCodes() {
		messages := []types.ContactMessage{}
		query := db.Query(&messages)
		query.Select(query.Created).
			WeekCode.Equals(weekCode).IP.Equals(ipKey).Created.Between(windowStart, nowTime)
		if err := query.Exec(); err != nil {
			return 0, err
		}
		total += int32(len(messages))
	}

	return total, nil
}

// makeContactEmailBody escapes every field: the whole body is attacker-controlled text, and it is
// read in a mail client that renders HTML.
func makeContactEmailBody(message *types.ContactMessage) string {
	companyLine := ""
	if len(message.Company) > 0 {
		companyLine = fmt.Sprintf(`<p><b>Empresa:</b> %v</p>`, html.EscapeString(message.Company))
	}

	return fmt.Sprintf(`<html>
	<head><meta http-equiv="Content-Type" content="text/html; charset=utf-8" /></head>
	<body style="font-family:system-ui,sans-serif;color:#1e293b;line-height:1.55">
		<h2 style="color:#4f46e5">Nuevo mensaje de contacto</h2>
		<p><b>Nombre:</b> %v</p>
		<p><b>Correo:</b> <a href="mailto:%v">%v</a></p>
		%v
		<p><b>Mensaje:</b></p>
		<p style="white-space:pre-wrap;border-left:3px solid #c7d2fe;padding-left:12px">%v</p>
		<p style="color:#64748b;font-size:13px">Enviado desde el formulario de contacto de %v.</p>
	</body>
</html>`,
		html.EscapeString(message.Name),
		html.EscapeString(message.Email), html.EscapeString(message.Email),
		companyLine,
		html.EscapeString(message.Message),
		html.EscapeString(core.Env.APP_URL))
}

// PostContactMessage stores one contact-form submission and notifies the configured inbox.
func PostContactMessage(req *core.HandlerArgs) core.HandlerResponse {
	body := struct {
		Name    string
		Email   string
		Company string
		Message string
	}{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	// Refused before anything is written: a message with no destination is only a row nobody will
	// ever read, and the form on /welcome already tells the visitor the destination is unset.
	if len(core.Env.CONTACT_EMAIL) == 0 {
		return req.MakeErr("El formulario de contacto no está configurado: falta contact.email en config.toml.")
	}

	name := strings.TrimSpace(body.Name)
	email := normalizeEmail(body.Email)
	company := strings.TrimSpace(body.Company)
	message := strings.TrimSpace(body.Message)

	if utf8.RuneCountInString(name) < contactMinNameRunes {
		return req.MakeErr("El nombre es necesario.")
	}
	if utf8.RuneCountInString(name) > contactMaxNameRunes {
		return req.MakeErr(fmt.Sprintf("El nombre no puede exceder %v caracteres.", contactMaxNameRunes))
	}
	if utf8.RuneCountInString(email) > contactMaxEmailRunes || !emailPattern.MatchString(email) {
		return req.MakeErr("El correo electrónico no posee un formato válido.")
	}
	if utf8.RuneCountInString(company) > contactMaxCompanyRunes {
		return req.MakeErr(fmt.Sprintf("La empresa no puede exceder %v caracteres.", contactMaxCompanyRunes))
	}
	if utf8.RuneCountInString(message) < contactMinMessageRunes {
		return req.MakeErr(fmt.Sprintf("El mensaje debe poseer al menos %v caracteres.", contactMinMessageRunes))
	}
	if utf8.RuneCountInString(message) > contactMaxMessageRunes {
		return req.MakeErr(fmt.Sprintf("El mensaje no puede exceder %v caracteres.", contactMaxMessageRunes))
	}

	ipKey, hasClientIP := req.ClientIPKey()
	if !hasClientIP {
		return req.MakeErr("No se pudo determinar el origen de la solicitud.")
	}

	// Counting and inserting have to happen as one step, so they run one caller at a time. The
	// queue is shallow on purpose: parallel submissions from one IP are the abuse pattern, so
	// refusing the extras quickly is the wanted behavior.
	//
	// Failing closed on an unreachable daemon is the point, as it is in sign-up: with no lock this
	// endpoint is an open relay into somebody's inbox, and a contact form is on nobody's critical
	// path.
	contactLock, lockErr := core.AcquireLock(context.Background(), core.ActionContactByIP, ipKey, 2)
	if lockErr != nil {
		return lockErr.Response(req)
	}
	defer contactLock.Release()

	recentMessages, err := countRecentMessagesFromIP(ipKey)
	if err != nil {
		return req.MakeErr("Error al verificar el origen de la solicitud.", err)
	}
	if recentMessages >= core.Env.CONTACT_MAX_MESSAGES_PER_IP {
		core.Log("PostContactMessage:: límite por IP alcanzado", ipKey, recentMessages)
		return req.MakeErrCode(fmt.Sprintf(
			"Se alcanzó el máximo de %d mensajes por %d minutos desde esta conexión.",
			core.Env.CONTACT_MAX_MESSAGES_PER_IP, core.Env.CONTACT_WINDOW_MINUTES), 429)
	}

	sequenceID, err := db.GetAutoincrementID("contact_message", 1)
	if err != nil {
		return req.MakeErr("Error al generar el ID del mensaje.", err)
	}

	nowTime := core.SUnixTime()
	newMessages := []types.ContactMessage{{
		WeekCode: weekCodeAt(core.Now()),
		IP:       ipKey,
		Created:  nowTime,
		ID:       sequenceID,
		Name:     name,
		Email:    email,
		Company:  company,
		Message:  message,
		Updated:  nowTime,
		Status:   contactStatusDelivered,
	}}

	// Persist first, deliver second — the opposite of sign-up, and for the opposite reason. There
	// the row is worthless without the email that carries the code, so a failed send must leave no
	// row behind; here the row IS the message, and losing it to an SMTP hiccup would lose what the
	// visitor wrote. Writing first also means a failed send still consumes the sender's budget,
	// which is what stops a broken mail server from turning into an unmetered endpoint.
	if err := db.Insert(&newMessages); err != nil {
		return req.MakeErr("Error al registrar el mensaje de contacto.", err)
	}
	storedMessage := newMessages[0]

	if err := core.SendEmail(core.Env.CONTACT_EMAIL, contactEmailSubject,
		makeContactEmailBody(&storedMessage)); err != nil {

		// The visitor is told the message arrived, because it did: it is stored, and status 2 is
		// what marks it as still needing to reach the inbox.
		core.Log("PostContactMessage:: no se pudo notificar el mensaje", storedMessage.ID, err)
		storedMessage.Status = contactStatusUndelivered
		storedMessage.Updated = core.SUnixTime()
		messageTable := db.TableOf[types.ContactMessage]()
		if err := db.Update(&[]types.ContactMessage{storedMessage},
			messageTable.Status, messageTable.Updated); err != nil {
			core.Log("PostContactMessage:: no se pudo marcar el mensaje como no entregado",
				storedMessage.ID, err)
		}
		return req.MakeResponse(map[string]any{"Received": true, "Notified": false})
	}

	core.Log("PostContactMessage:: mensaje recibido", storedMessage.ID, "de", email)
	return req.MakeResponse(map[string]any{"Received": true, "Notified": true})
}
