package security

import (
	"app/cloud"
	configTypes "app/config/types"
	"app/core"
	coretypes "app/core/types"
	"app/db"
	"app/security/types"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

// Public registration: /welcome asks for an email, we mail an 8-digit code plus a link, and the
// verified request is then exchanged for a company and its admin user. Every endpoint here is
// public ("p-" prefix), which in this backend also means it bypasses the credit rate limiter
// (main-handlers.go). The throttling therefore lives in these handlers: one live request per
// email address, and a capped number of code attempts per request.

const (
	// signUpRandomFactor reserves the 6 low digits of the ID for randomness, so knowing one
	// request ID tells you nothing about the neighbouring ones. The digits above it are a global
	// sequence: the week is deliberately NOT encoded, because a request lives 2 hours and so can
	// only ever be in the current or the previous week partition, which findSignUpRequestByID
	// simply searches. A global (rather than per-week) sequence is what makes that search
	// unambiguous — the same ID cannot be minted in two different weeks.
	signUpRandomFactor = int64(1_000_000)
	// Two hours, expressed in SUnixTime units (1 unit = 2 seconds).
	signUpExpirySUnits = int32(3600)
	// Two minutes between two deliveries for the same request, so the endpoint cannot be used to
	// hammer an inbox. Within a live request the code is resent unchanged, never rotated, so a
	// code taken from an earlier email keeps working.
	signUpResendCooldownSUnits = int32(60)
	// A wrong code past this count burns the request instead of allowing an endless sweep.
	signUpMaxAttempts = int8(5)

	signUpStatusCancelled = int8(0)
	signUpStatusSent      = int8(1)
	signUpStatusVerified  = int8(2)
	signUpStatusCompleted = int8(3)
)

const signUpEmailSubject = "Genix — código de verificación"

var signUpEmailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)

// normalizeSignUpEmail is what gets stored and compared, so "A@B.com" and "a@b.com " are the
// same account for the "one company per email" rule.
func normalizeSignUpEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// randomDigits returns a uniformly random integer with exactly `digits` decimal places of range
// (0 .. 10^digits-1). crypto/rand and not math/rand: this backs a verification code.
func randomDigits(digits int) (int64, error) {
	upperBound := big.NewInt(1)
	for range digits {
		upperBound.Mul(upperBound, big.NewInt(10))
	}
	value, err := rand.Int(rand.Reader, upperBound)
	if err != nil {
		return 0, core.Err("Error al generar un valor aleatorio:", err)
	}
	return value.Int64(), nil
}

// signUpWeekCodeAt is the [year][week] partition code for a moment in time: year*100 + isoWeek
// - 200000, so 2026-W32 is 2632. The helper is called with an explicit day (never 0) because
// MakeSemanaFromFechaUnix memoizes by its argument, and 0 would pin "the current week" forever
// in a long-running server process.
func signUpWeekCodeAt(moment time.Time) int32 {
	return int32(core.MakeSemanaFromFechaUnix(core.TimeToFechaUnix(moment), true).Code)
}

// signUpSearchWeekCodes are the partitions a lookup has to touch. A request created late on a
// Sunday is still live on Monday, when the current week code has already moved on.
func signUpSearchWeekCodes() []int32 {
	now := time.Now()
	currentWeekCode := signUpWeekCodeAt(now)
	previousWeekCode := signUpWeekCodeAt(now.AddDate(0, 0, -7))
	if previousWeekCode == currentWeekCode {
		return []int32{currentWeekCode}
	}
	return []int32{currentWeekCode, previousWeekCode}
}

// makeSignUpRequestID composes sequence | random. The sequence comes from the ORM's counter under
// a single global key, so two concurrent requests can never collide no matter which week they land
// in; the random tail is what stops the IDs from being walked one by one.
func makeSignUpRequestID() (int64, error) {
	sequence, err := db.GetAutoincrementID("signup", 1)
	if err != nil {
		return 0, core.Err("Error al generar el ID de la solicitud:", err)
	}
	randomSuffix, err := randomDigits(6)
	if err != nil {
		return 0, err
	}
	return sequence*signUpRandomFactor + randomSuffix, nil
}

func isSignUpRequestExpired(request *types.SignUpRequest) bool {
	return core.SUnixTime()-request.Created > signUpExpirySUnits
}

// findSignUpRequestByID resolves the bare "req" value from the email link. The ID carries no
// partition, so both week partitions a live request can be in are searched — a request expires
// after 2 hours, so it is always in the current week or the one before it.
func findSignUpRequestByID(requestID int64) (*types.SignUpRequest, error) {
	if requestID <= 0 {
		return nil, nil
	}

	for _, weekCode := range signUpSearchWeekCodes() {
		requests := []types.SignUpRequest{}
		query := db.Query(&requests)
		query.WeekCode.Equals(weekCode).ID.Equals(requestID).Limit(1)
		if err := query.Exec(); err != nil {
			return nil, err
		}
		if len(requests) > 0 {
			return &requests[0], nil
		}
	}

	return nil, nil
}

// findLatestSignUpRequestByEmail returns the most recent request for an address across the
// partitions a live request can be in.
func findLatestSignUpRequestByEmail(email string) (*types.SignUpRequest, error) {
	var latestRequest *types.SignUpRequest

	for _, weekCode := range signUpSearchWeekCodes() {
		requests := []types.SignUpRequest{}
		query := db.Query(&requests)
		query.WeekCode.Equals(weekCode).Email.Equals(email)
		if err := query.Exec(); err != nil {
			return nil, err
		}
		for index := range requests {
			if latestRequest == nil || requests[index].Created > latestRequest.Created {
				latestRequest = &requests[index]
			}
		}
	}

	return latestRequest, nil
}

// countRecentEmailsFromIP returns how many distinct addresses this client has already asked to
// register inside the window, and whether the address at hand is one of them. Retrying the same
// email must not consume budget — the resend cooldown already governs that — so only a genuinely
// new address is measured against the ceiling.
//
// Correct only while the caller holds the per-IP lock: without it, parallel Lambdas all read the
// same count, all conclude they are under the limit, and all insert.
func countRecentEmailsFromIP(ipKey int64, email string) (int32, bool, error) {
	windowStart := core.SUnixTime() - (core.Env.SIGNUP_WINDOW_MINUTES * 30) // 1 SUnixTime unit = 2s
	distinctEmails := map[string]bool{}
	emailAlreadyUsed := false

	for _, weekCode := range signUpSearchWeekCodes() {
		requests := []types.SignUpRequest{}
		query := db.Query(&requests)
		query.WeekCode.Equals(weekCode).IP.Equals(ipKey)
		if err := query.Exec(); err != nil {
			return 0, false, err
		}
		for index := range requests {
			if requests[index].Created < windowStart {
				continue
			}
			distinctEmails[requests[index].Email] = true
			if requests[index].Email == email {
				emailAlreadyUsed = true
			}
		}
	}

	return int32(len(distinctEmails)), emailAlreadyUsed, nil
}

// companyExistsWithEmail enforces "one company per email". The lookup rides on the global index
// declared on Company.Email; companies have no tenant partition, so it cannot be a local one.
func companyExistsWithEmail(email string) (bool, error) {
	companies := []configTypes.Company{}
	var err error
	if cloud.IsDataMirrorEnabled() {
		err = cloud.Select(&companies).Where("email").Equals(email).Exec()
	} else {
		query := db.Query(&companies)
		query.Email.Equals(email).Limit(1)
		err = query.Exec()
	}
	if err != nil {
		return false, err
	}
	return len(companies) > 0, nil
}

// resolveVerifiedSignUpRequest loads a request and checks the code, counting the failure when it
// does not match. Shared by the verification step and the company-creation step so the code is
// re-proved on every public call instead of trusting a client-held "already verified" flag.
func resolveVerifiedSignUpRequest(requestID int64, code string) (*types.SignUpRequest, error) {
	request, err := findSignUpRequestByID(requestID)
	if err != nil {
		return nil, core.Err("Error al obtener la solicitud de registro:", err)
	}
	if request == nil {
		return nil, core.Err("No se encontró la solicitud de registro.")
	}
	if request.Status == signUpStatusCancelled {
		return nil, core.Err("Esta solicitud de registro fue anulada. Inicie una nueva.")
	}
	if request.Status == signUpStatusCompleted {
		return nil, core.Err("Esta solicitud de registro ya fue utilizada para crear una empresa.")
	}
	if isSignUpRequestExpired(request) {
		return nil, core.Err("La solicitud de registro expiró. Solicite un nuevo correo.")
	}

	if request.Code != strings.TrimSpace(code) {
		request.Attempts++
		// Burning the request on the last attempt is what makes the 8-digit code safe on an
		// endpoint with no platform rate limiter.
		if request.Attempts >= signUpMaxAttempts {
			request.Status = signUpStatusCancelled
		}
		request.Updated = core.SUnixTime()
		requestTable := db.TableOf[types.SignUpRequest]()
		if err := db.Update(&[]types.SignUpRequest{*request},
			requestTable.Attempts, requestTable.Status, requestTable.Updated); err != nil {
			core.Log("resolveVerifiedSignUpRequest:: no se pudo registrar el intento fallido:", err)
		}
		if request.Status == signUpStatusCancelled {
			return nil, core.Err("Código incorrecto. Se agotaron los intentos: inicie un nuevo registro.")
		}
		return nil, core.Err(fmt.Sprintf("Código incorrecto. Le quedan %v intentos.",
			signUpMaxAttempts-request.Attempts))
	}

	return request, nil
}

func makeSignUpEmailBody(requestID int64, code string) string {
	verificationLink := fmt.Sprintf("%v/welcome?req=%v&code=%v", core.Env.APP_URL, requestID, code)

	return fmt.Sprintf(`<html>
	<head><meta http-equiv="Content-Type" content="text/html; charset=utf-8" /></head>
	<body style="font-family:system-ui,sans-serif;color:#1e293b;line-height:1.55">
		<h2 style="color:#4f46e5">Registro en Genix</h2>
		<p>Su código de verificación es:</p>
		<p style="font-size:28px;font-weight:700;letter-spacing:6px;color:#4f46e5">%v</p>
		<p>También puede continuar el registro haciendo clic en este enlace:</p>
		<p><a href="%v" style="color:#4f46e5">%v</a></p>
		<p style="color:#64748b;font-size:13px">El enlace y el código caducan en 2 horas.
		Si usted no solicitó este registro, ignore este correo.</p>
	</body>
</html>`, code, verificationLink, verificationLink)
}

// PostSignUpRequest is step 1: it mails a verification code to an address that does not yet own
// a company.
func PostSignUpRequest(req *core.HandlerArgs) core.HandlerResponse {
	body := struct{ Email string }{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	email := normalizeSignUpEmail(body.Email)
	if !signUpEmailPattern.MatchString(email) {
		return req.MakeErr("El correo electrónico no posee un formato válido.")
	}
	if len(core.Env.APP_URL) == 0 {
		return req.MakeErr("El registro público no está configurado: falta app_url en config.toml.")
	}

	ipKey, hasClientIP := req.ClientIPKey()
	if !hasClientIP {
		return req.MakeErr("No se pudo determinar el origen de la solicitud.")
	}

	// Everything below reads the state of this IP and then writes to it, so it has to run one
	// caller at a time. The queue is deliberately shallow: parallel requests from one IP are the
	// abuse pattern, so refusing the extras fast is the wanted behavior, not a degradation.
	//
	// Lease must outlast the whole critical section, SMTP included, or the daemon hands the key
	// to the next caller while this one is still working — the exact race the lock exists to
	// prevent. Worst case here is core.SendEmail's 4s connect + 6s send plus the queries, so 30s
	// is roughly triple the bound. Raising the mailer timeouts means raising this.
	signUpLock, err := core.AcquireLock(context.Background(), core.ActionSignUpByIP, ipKey, core.LockOptions{
		MaxWaiters: 2,
		Wait:       5 * time.Second,
		Lease:      30 * time.Second,
	})
	if err != nil {
		if errors.Is(err, core.ErrLockBusy) {
			return req.MakeErrCode("Demasiadas solicitudes simultáneas. Intente nuevamente.", 429)
		}
		// Fails closed, unlike the credit limiter: with the daemon down this endpoint would be an
		// open relay for verification emails, and registration is on nobody's critical path.
		core.Log("lock service unavailable, refusing sign-up::", err)
		return req.MakeErrCode("El servicio de registro no está disponible.", 503)
	}
	defer signUpLock.Release()

	recentEmails, emailAlreadyUsed, err := countRecentEmailsFromIP(ipKey, email)
	if err != nil {
		return req.MakeErr("Error al verificar el origen de la solicitud.", err)
	}
	if !emailAlreadyUsed && recentEmails >= core.Env.SIGNUP_MAX_EMAILS_PER_IP {
		core.Log("PostSignUpRequest:: límite por IP alcanzado", ipKey, recentEmails)
		return req.MakeErrCode(fmt.Sprintf(
			"Se alcanzó el máximo de %d registros por %d minutos desde esta conexión.",
			core.Env.SIGNUP_MAX_EMAILS_PER_IP, core.Env.SIGNUP_WINDOW_MINUTES), 429)
	}

	companyExists, err := companyExistsWithEmail(email)
	if err != nil {
		return req.MakeErr("Error al verificar el correo electrónico.", err)
	}
	if companyExists {
		return req.MakeErr("Ya existe una empresa registrada con este correo electrónico.")
	}

	latestRequest, err := findLatestSignUpRequestByEmail(email)
	if err != nil {
		return req.MakeErr("Error al consultar las solicitudes de registro.", err)
	}

	nowTime := core.SUnixTime()

	// A live request is reused rather than replaced, so the address keeps one ID and one code no
	// matter how many times the email is asked for.
	if latestRequest != nil && !isSignUpRequestExpired(latestRequest) &&
		(latestRequest.Status == signUpStatusSent || latestRequest.Status == signUpStatusVerified) {

		elapsedSUnits := nowTime - latestRequest.LastSent
		if elapsedSUnits < signUpResendCooldownSUnits {
			core.Log("PostSignUpRequest:: reenvío en espera para la solicitud", latestRequest.ID)
			return req.MakeResponse(map[string]any{
				"RequestID":         latestRequest.ID,
				"Sent":              false,
				"Verified":          latestRequest.Status == signUpStatusVerified,
				"SentAt":            latestRequest.LastSent,
				"RetryAfterSeconds": (signUpResendCooldownSUnits - elapsedSUnits) * 2,
			})
		}

		if err := core.SendEmail(email, signUpEmailSubject,
			makeSignUpEmailBody(latestRequest.ID, latestRequest.Code)); err != nil {
			return req.MakeErr("No se pudo enviar el correo de verificación.", err)
		}

		latestRequest.LastSent = nowTime
		latestRequest.Updated = nowTime
		requestTable := db.TableOf[types.SignUpRequest]()
		if err := db.Update(&[]types.SignUpRequest{*latestRequest},
			requestTable.LastSent, requestTable.Updated); err != nil {
			return req.MakeErr("Error al registrar el reenvío del correo.", err)
		}

		core.Log("PostSignUpRequest:: correo reenviado para la solicitud", latestRequest.ID)
		return req.MakeResponse(map[string]any{
			"RequestID":         latestRequest.ID,
			"Sent":              true,
			"Verified":          latestRequest.Status == signUpStatusVerified,
			"SentAt":            nowTime,
			"RetryAfterSeconds": signUpResendCooldownSUnits * 2,
		})
	}

	weekCode := signUpWeekCodeAt(time.Now())
	requestID, err := makeSignUpRequestID()
	if err != nil {
		return req.MakeErr(err)
	}
	codeNumber, err := randomDigits(8)
	if err != nil {
		return req.MakeErr(err)
	}
	code := fmt.Sprintf("%08d", codeNumber)

	// Deliver first, persist second. A row written before a failed send would be a request whose
	// code nobody ever received, and it would then block every retry until it expired.
	if err := core.SendEmail(email, signUpEmailSubject, makeSignUpEmailBody(requestID, code)); err != nil {
		return req.MakeErr("No se pudo enviar el correo de verificación.", err)
	}

	newRequests := []types.SignUpRequest{{
		WeekCode: weekCode,
		ID:       requestID,
		Email:    email,
		Code:     code,
		IP:       ipKey,
		Created:  nowTime,
		LastSent: nowTime,
		Updated:  nowTime,
		Status:   signUpStatusSent,
	}}
	if err := db.Insert(&newRequests); err != nil {
		return req.MakeErr("Error al registrar la solicitud de registro.", err)
	}

	core.Log("PostSignUpRequest:: solicitud creada", requestID, "semana", weekCode)

	return req.MakeResponse(map[string]any{
		"RequestID":         requestID,
		"Sent":              true,
		"Verified":          false,
		"SentAt":            nowTime,
		"RetryAfterSeconds": signUpResendCooldownSUnits * 2,
	})
}

// PostSignUpVerify is step 1b: it exchanges the emailed code for a verified request.
func PostSignUpVerify(req *core.HandlerArgs) core.HandlerResponse {
	body := struct {
		RequestID int64
		Code      string
	}{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}
	if body.RequestID <= 0 || len(body.Code) == 0 {
		return req.MakeErr("Faltan la solicitud o el código de verificación.")
	}

	request, err := resolveVerifiedSignUpRequest(body.RequestID, body.Code)
	if err != nil {
		return req.MakeErr(err)
	}

	if request.Status != signUpStatusVerified {
		request.Status = signUpStatusVerified
		request.Updated = core.SUnixTime()
		requestTable := db.TableOf[types.SignUpRequest]()
		if err := db.Update(&[]types.SignUpRequest{*request},
			requestTable.Status, requestTable.Updated); err != nil {
			return req.MakeErr("Error al confirmar la verificación del correo.", err)
		}
	}

	core.Log("PostSignUpVerify:: solicitud verificada", request.ID)
	return req.MakeResponse(map[string]any{"RequestID": request.ID, "Email": request.Email})
}

// PostSignUpCompany is step 2: it creates the company and its admin user, and answers with the
// same payload as a normal login so the browser is authenticated for step 3 ("Initial Data"),
// which then reuses the existing private initial-data endpoint.
func PostSignUpCompany(req *core.HandlerArgs) core.HandlerResponse {
	body := struct {
		RequestID      int64
		Code           string
		CompanyName    string
		Address        string
		RUC            string
		AdminFirstName string
		AdminLastName  string
		AdminPassword  string
		CipherKey      string
	}{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	body.CompanyName = strings.TrimSpace(body.CompanyName)
	body.AdminFirstName = strings.TrimSpace(body.AdminFirstName)
	body.AdminLastName = strings.TrimSpace(body.AdminLastName)

	if len(body.CompanyName) < 5 {
		return req.MakeErr("El nombre de la empresa debe poseer al menos 5 caracteres.")
	}
	if len(body.AdminPassword) < 6 {
		return req.MakeErr("La contraseña debe poseer al menos 6 caracteres.")
	}
	if len(body.CipherKey) == 0 {
		return req.MakeErr("El CipherKey es necesario.")
	}

	request, err := resolveVerifiedSignUpRequest(body.RequestID, body.Code)
	if err != nil {
		return req.MakeErr(err)
	}
	if request.Status != signUpStatusVerified {
		return req.MakeErr("Debe verificar su correo electrónico antes de crear la empresa.")
	}

	// Re-checked here and not only in step 1: the two calls are minutes apart and the endpoint is
	// public, so the address could have been claimed in between.
	companyExists, err := companyExistsWithEmail(request.Email)
	if err != nil {
		return req.MakeErr("Error al verificar el correo electrónico.", err)
	}
	if companyExists {
		return req.MakeErr("Ya existe una empresa registrada con este correo electrónico.")
	}

	nowTime := core.SUnixTime()
	newCompanies := []configTypes.Company{{
		Name:      body.CompanyName,
		LegalName: body.CompanyName,
		RUC:       strings.TrimSpace(body.RUC),
		Address:   strings.TrimSpace(body.Address),
		Email:     request.Email,
		// The address is proven at this point: the code only reached whoever controls the inbox.
		EmailVerified: 1,
		FormApiKey:    core.MakeRandomBase36String(18),
		Updated:       nowTime,
		Status:        1,
	}}
	if err := db.Insert(&newCompanies); err != nil {
		return req.MakeErr("Error al crear la empresa.", err)
	}
	newCompany := newCompanies[0]
	core.Log("PostSignUpCompany:: empresa creada", newCompany.ID, newCompany.Name)

	if cloud.IsDataMirrorEnabled() {
		if err := cloud.Insert([]configTypes.Company{newCompany}); err != nil {
			return req.MakeErr("Error al guardar la empresa en el espejo cloud.", err)
		}
	}

	// The ORM's sequence is partitioned by CompanyID, so the first user of a brand-new company
	// gets ID 1 — which is the ID MakeUsuarioResponse grants every declared access to, and whose
	// login is always "admin" (PostUsuario enforces the same rule on later edits).
	// FirstName falls back to "admin" because it is optional here but required when the profile is
	// saved from the users page, which would otherwise reject the record it just loaded.
	passwordHash := core.FnvHashString64(core.Env.SECRET_PHRASE+body.AdminPassword, -1, 20)
	newUsers := []coretypes.User{{
		CompanyID:    newCompany.ID,
		User:         "admin",
		FirstName:    core.If(len(body.AdminFirstName) > 0, body.AdminFirstName, "admin"),
		LastName:     body.AdminLastName,
		Email:        request.Email,
		PasswordHash: passwordHash,
		Created:      nowTime,
		Updated:      nowTime,
		Status:       1,
	}}
	if err := db.Insert(&newUsers); err != nil {
		return req.MakeErr("Error al crear el usuario administrador.", err)
	}
	newUser := newUsers[0]
	core.Log("PostSignUpCompany:: usuario administrador creado", newUser.ID, "en la empresa", newCompany.ID)

	if cloud.IsDataMirrorEnabled() {
		if err := cloud.Insert([]coretypes.User{newUser}); err != nil {
			return req.MakeErr("Error al guardar el usuario en el espejo cloud.", err)
		}
	}

	request.Status = signUpStatusCompleted
	request.CompanyID = newCompany.ID
	request.UserID = newUser.ID
	request.Updated = nowTime
	requestTable := db.TableOf[types.SignUpRequest]()
	if err := db.Update(&[]types.SignUpRequest{*request},
		requestTable.Status, requestTable.CompanyID, requestTable.UserID,
		requestTable.Updated); err != nil {
		// The company already exists, so failing the whole call here would leave the caller unable
		// to log in to something that was created. Log and continue.
		core.Log("PostSignUpCompany:: no se pudo cerrar la solicitud", request.ID, err)
	}

	loginResponse, err := MakeUsuarioResponse(newUser, body.CipherKey)
	if err != nil {
		return req.MakeErr(err)
	}
	loginResponse["CompanyName"] = newCompany.Name

	return req.MakeResponse(loginResponse)
}
