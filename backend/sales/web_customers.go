package sales

import (
	"app/cloud"
	config "app/config/types"
	"app/core"
	"app/db"
	"app/sales/types"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// The web store's customer accounts: emailed 6-digit codes, passwordless login, the customer's
// profile and saved addresses. Every route is public ("p-"): the back-office gate does not apply,
// so each one checks the customer token itself (core.CheckWebCustomer) and throttles its own
// abuse, like security/signup.go does. See docs/WEB_CHECKOUT_PLAN.md §5.

const (
	// A code lives 30 minutes (SUnixTime units are 2 seconds).
	webCustomerCodeLifeSUnits = int32(900)
	// One minute between two emails for the same address; a resend repeats the live code.
	webCustomerCodeResendSUnits = int32(30)
	// A wrong code past this count burns it, so a 6-digit code cannot be swept.
	webCustomerCodeMaxAttempts = int8(5)
	// Per client IP and hour: distinct addresses mailed, and customers created without a code.
	webCustomerWindowSUnits      = int32(1800)
	webCustomerMaxEmailsPerIP    = 10
	webCustomerMaxCreatedPerIP   = 5
	webCustomerMinPasswordLength = 6
)

var webCustomerEmailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)
var webCustomerPhonePattern = regexp.MustCompile(`^\+?[0-9 ]{6,15}$`)

// webStoreCompanyID is the store a public request is for: the storefront sends ?cid= (or the
// company-id every Env.makeRoute call appends).
func webStoreCompanyID(req *core.HandlerArgs) int32 {
	return int32(core.Coalesce(req.GetQueryInt("cid"), req.GetQueryInt("company-id")))
}

// webCustomerProfile is what a shopper types about themselves, on Regístrate, Mis Datos and the
// cart's Datos Envío.
type webCustomerProfile struct {
	FirstName      string
	LastName       string
	Phone          string
	DocumentNumber string
}

// validateWebCustomerProfile trims the profile and checks it. The phone is required only when the
// company turned on flag 11 "Requerir número de contacto".
func validateWebCustomerProfile(profile *webCustomerProfile, companyFlags []int16) error {
	profile.FirstName = strings.TrimSpace(profile.FirstName)
	profile.LastName = strings.TrimSpace(profile.LastName)
	profile.Phone = strings.TrimSpace(profile.Phone)
	profile.DocumentNumber = strings.TrimSpace(profile.DocumentNumber)
	if len(profile.FirstName) < 2 || len(profile.LastName) < 2 {
		return core.Err("Los nombres y apellidos deben tener al menos 2 caracteres.")
	}
	if profile.Phone == "" && core.HasCompanyFlag(companyFlags, types.CompanyFlagRequireContactNumber) {
		return core.Err("Debe ingresar un número de celular.")
	}
	if profile.Phone != "" && !webCustomerPhonePattern.MatchString(profile.Phone) {
		return core.Err("El número de celular no es válido.")
	}
	if profile.DocumentNumber != "" && !isValidDNI(profile.DocumentNumber) {
		return core.Err("El DNI debe tener 8 dígitos.")
	}
	return nil
}

func isValidDNI(documentNumber string) bool {
	if len(documentNumber) != 8 {
		return false
	}
	for _, character := range documentNumber {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func findWebCustomerByEmail(companyID int32, email string) (*types.WebCustomer, error) {
	customers := []types.WebCustomer{}
	query := db.Query(&customers)
	query.Select().CompanyID.Equals(companyID).Email.Equals(email).Limit(1)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al buscar el cliente:", err)
	}
	if len(customers) == 0 {
		return nil, nil
	}
	return &customers[0], nil
}

func findWebCustomerByID(companyID, customerID int32) (*types.WebCustomer, error) {
	customers := []types.WebCustomer{}
	query := db.Query(&customers)
	query.Select().CompanyID.Equals(companyID).ID.Equals(customerID).Limit(1)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener el cliente:", err)
	}
	if len(customers) == 0 || customers[0].Status == 0 {
		return nil, core.Err("No se encontró el cliente.")
	}
	return &customers[0], nil
}

func listWebCustomerAddresses(companyID, customerID int32) ([]types.WebCustomerAddress, error) {
	addresses := []types.WebCustomerAddress{}
	query := db.Query(&addresses)
	query.Select().CompanyID.Equals(companyID).CustomerID.Equals(customerID)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener las direcciones:", err)
	}
	return slices.DeleteFunc(addresses, func(address types.WebCustomerAddress) bool { return address.Status == 0 }), nil
}

// requireWebCustomer resolves the session of a customer route to the customer's record.
func requireWebCustomer(req *core.HandlerArgs) (int32, *types.WebCustomer, error) {
	companyID := webStoreCompanyID(req)
	if companyID <= 0 {
		return 0, nil, core.Err("No se indicó la tienda.")
	}
	token, err := core.CheckWebCustomer(req, companyID)
	if err != nil {
		return 0, nil, err
	}
	customer, err := findWebCustomerByID(companyID, token.CustomerID)
	if err != nil {
		return 0, nil, err
	}
	return companyID, customer, nil
}

// makeWebCustomerSession is the answer of every route that logs a customer in: a fresh token, the
// customer as they may see it, and their addresses.
func makeWebCustomerSession(req *core.HandlerArgs, customer *types.WebCustomer) core.HandlerResponse {
	token, err := core.MakeWebCustomerToken(customer.CompanyID, customer.ID)
	if err != nil {
		return req.MakeErr(err)
	}
	addresses, err := listWebCustomerAddresses(customer.CompanyID, customer.ID)
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(map[string]any{
		"Token":       token,
		"Customer":    customer.PublicWebCustomer(),
		"HasPassword": customer.HasPassword(),
		"Addresses":   addresses,
	})
}

func randomSixDigitCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", core.Err("Error al generar el código:", err)
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func makeWebCustomerCodeEmail(storeName, code string) string {
	return fmt.Sprintf(`<html>
	<head><meta http-equiv="Content-Type" content="text/html; charset=utf-8" /></head>
	<body style="font-family:system-ui,sans-serif;color:#1e293b;line-height:1.55">
		<h2 style="color:#4f46e5">%v</h2>
		<p>Su código de verificación es:</p>
		<p style="font-size:28px;font-weight:700;letter-spacing:6px;color:#4f46e5">%v</p>
		<p style="color:#64748b;font-size:13px">El código caduca en 30 minutos.
		Si usted no lo solicitó, ignore este correo.</p>
	</body>
</html>`, storeName, code)
}

// webCustomerCodeResult is what the storefront needs to show after asking for a code.
type webCustomerCodeResult struct {
	Sent              bool
	RetryAfterSeconds int32
}

// sendWebCustomerCode mails the address its live code, minting one when there is none. Within a
// live code the same code is resent, never rotated, so an earlier email keeps working.
func sendWebCustomerCode(req *core.HandlerArgs, companyID int32, email string) (webCustomerCodeResult, error) {
	ipKey, hasClientIP := req.ClientIPKey()
	if !hasClientIP {
		return webCustomerCodeResult{}, core.Err("No se pudo determinar el origen de la solicitud.")
	}
	// The count below is read and then written, so it runs one caller per IP at a time. Fail
	// closed: without the lock this endpoint is an open relay for emails.
	ipLock, lockErr := core.AcquireLock(context.Background(), core.ActionWebCustomerByIP, ipKey, 2)
	if lockErr != nil {
		return webCustomerCodeResult{}, lockErr
	}
	defer ipLock.Release()

	codes := []types.WebCustomerCode{}
	query := db.Query(&codes)
	query.Select().CompanyID.Equals(companyID).Email.Equals(email).Limit(1)
	if err := query.Exec(); err != nil {
		return webCustomerCodeResult{}, core.Err("Error al obtener el código:", err)
	}

	nowTime := core.SUnixTime()
	codeRow := types.WebCustomerCode{CompanyID: companyID, Email: email}
	if len(codes) > 0 {
		codeRow = codes[0]
	}
	isLiveCode := codeRow.ID > 0 && codeRow.Status == types.WebCustomerCodeSent &&
		nowTime-codeRow.Created < webCustomerCodeLifeSUnits && codeRow.Attempts < webCustomerCodeMaxAttempts

	if isLiveCode && nowTime-codeRow.LastSent < webCustomerCodeResendSUnits {
		return webCustomerCodeResult{RetryAfterSeconds: (webCustomerCodeResendSUnits - (nowTime - codeRow.LastSent)) * 2}, nil
	}

	if !isLiveCode {
		// Only a new address counts against the IP: resending to the same one is paced above.
		recentEmails, err := countRecentWebCodeEmailsFromIP(companyID, ipKey, email)
		if err != nil {
			return webCustomerCodeResult{}, err
		}
		if recentEmails >= webCustomerMaxEmailsPerIP {
			return webCustomerCodeResult{}, core.Err("Se alcanzó el máximo de correos de verificación desde esta conexión. Intente en una hora.")
		}
		code, err := randomSixDigitCode()
		if err != nil {
			return webCustomerCodeResult{}, err
		}
		codeRow.Code, codeRow.Created, codeRow.Attempts, codeRow.Status = code, nowTime, 0, types.WebCustomerCodeSent
	}

	companyConfig, err := cloud.LoadCompanyConfig(companyID)
	if err != nil {
		return webCustomerCodeResult{}, core.Err("Error al leer la configuración de la tienda:", err)
	}
	// Deliver first, persist second: a row for a code nobody received would block the retries.
	if err := core.SendEmail(companyID, core.EmailMessage{
		To:       []string{email},
		FromName: companyConfig.Company.TradeName,
		Subject:  companyConfig.Company.TradeName + " — código de verificación",
		HTMLBody: makeWebCustomerCodeEmail(companyConfig.Company.TradeName, codeRow.Code),
	}); err != nil {
		return webCustomerCodeResult{}, core.Err("No se pudo enviar el correo de verificación:", err)
	}

	codeRow.LastSent, codeRow.Updated, codeRow.IP = nowTime, nowTime, ipKey
	rows := []types.WebCustomerCode{codeRow}
	if codeRow.ID > 0 {
		// Email is the unique index key: the ORM refuses to rewrite it on an existing row.
		err = db.UpdateExclude(&rows, db.TableOf[types.WebCustomerCode]().Email)
	} else {
		err = db.Insert(&rows)
	}
	if err != nil {
		return webCustomerCodeResult{}, core.Err("Error al registrar el código:", err)
	}
	return webCustomerCodeResult{Sent: true, RetryAfterSeconds: webCustomerCodeResendSUnits * 2}, nil
}

// countRecentWebCodeEmailsFromIP counts the other addresses this IP had a code mailed to within
// the window. Correct only under the per-IP lock.
func countRecentWebCodeEmailsFromIP(companyID int32, ipKey int64, email string) (int, error) {
	codes := []types.WebCustomerCode{}
	query := db.Query(&codes)
	query.Select(query.Email, query.LastSent).CompanyID.Equals(companyID).IP.Equals(ipKey)
	if err := query.Exec(); err != nil {
		return 0, core.Err("Error al verificar el origen de la solicitud:", err)
	}
	windowStart := core.SUnixTime() - webCustomerWindowSUnits
	count := 0
	for _, code := range codes {
		if code.LastSent >= windowStart && code.Email != email {
			count++
		}
	}
	return count, nil
}

// consumeWebCustomerCode checks a typed code and burns it on success, so it opens one session.
func consumeWebCustomerCode(companyID int32, email, typedCode string) error {
	codes := []types.WebCustomerCode{}
	query := db.Query(&codes)
	query.Select().CompanyID.Equals(companyID).Email.Equals(email).Limit(1)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener el código:", err)
	}
	if len(codes) == 0 || codes[0].Status != types.WebCustomerCodeSent {
		return core.Err("No hay un código vigente para este correo. Solicite uno nuevo.")
	}
	codeRow := codes[0]
	if core.SUnixTime()-codeRow.Created > webCustomerCodeLifeSUnits || codeRow.Attempts >= webCustomerCodeMaxAttempts {
		return core.Err("El código expiró. Solicite uno nuevo.")
	}

	codeTable := db.TableOf[types.WebCustomerCode]()
	codeRow.Updated = core.SUnixTime()
	if codeRow.Code != strings.TrimSpace(typedCode) {
		codeRow.Attempts++
		if err := db.Update(&[]types.WebCustomerCode{codeRow}, codeTable.Attempts, codeTable.Updated); err != nil {
			core.Log("consumeWebCustomerCode:: no se pudo registrar el intento fallido", err)
		}
		if codeRow.Attempts >= webCustomerCodeMaxAttempts {
			return core.Err("Código incorrecto. Se agotaron los intentos: solicite un nuevo código.")
		}
		return core.Err(fmt.Sprintf("Código incorrecto. Le quedan %v intentos.", webCustomerCodeMaxAttempts-codeRow.Attempts))
	}

	codeRow.Status = types.WebCustomerCodeUsed
	if err := db.Update(&[]types.WebCustomerCode{codeRow}, codeTable.Status, codeTable.Updated); err != nil {
		return core.Err("Error al confirmar el código:", err)
	}
	return nil
}

// createWebCustomer inserts a customer, or returns the one a concurrent request created for the
// same email: the unique index makes the insert fail, and the rerun finds the winner.
func createWebCustomer(companyID int32, email string, profile webCustomerProfile,
	emailConfirmed bool, ipKey int64) (*types.WebCustomer, error) {

	var customer *types.WebCustomer
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		existing, err := findWebCustomerByEmail(companyID, email)
		if err != nil {
			return err
		}
		if existing != nil {
			customer = existing
			return nil
		}
		nowTime := core.SUnixTime()
		newCustomers := []types.WebCustomer{{
			CompanyID: companyID, Email: email,
			FirstName: profile.FirstName, LastName: profile.LastName, Phone: profile.Phone,
			DocumentNumber: profile.DocumentNumber, EmailConfirmed: emailConfirmed,
			CreatedIP: ipKey, Status: 1, Created: nowTime, Updated: nowTime,
		}}
		if err := db.InsertIn(transaction, &newCustomers); err != nil {
			return core.Err("Error al registrar el cliente:", err)
		}
		customer = &newCustomers[0]
		return nil
	})
	return customer, err
}

// confirmAndFillWebCustomer marks the email confirmed (a code was just typed) and fills the
// profile fields the customer had left empty. A filled field is never overwritten from the cart:
// only the customer's own Mis Datos changes it.
func confirmAndFillWebCustomer(customer *types.WebCustomer, profile webCustomerProfile) error {
	customer.EmailConfirmed = true
	customer.FirstName = core.If(customer.FirstName == "", profile.FirstName, customer.FirstName)
	customer.LastName = core.If(customer.LastName == "", profile.LastName, customer.LastName)
	customer.Phone = core.If(customer.Phone == "", profile.Phone, customer.Phone)
	customer.DocumentNumber = core.If(customer.DocumentNumber == "", profile.DocumentNumber, customer.DocumentNumber)
	customer.Updated = core.SUnixTime()
	customerTable := db.TableOf[types.WebCustomer]()
	return db.Update(&[]types.WebCustomer{*customer}, customerTable.EmailConfirmed, customerTable.FirstName,
		customerTable.LastName, customerTable.Phone, customerTable.DocumentNumber, customerTable.Updated)
}

// validateWebCustomerAddress trims an address and checks it names a district.
func validateWebCustomerAddress(address *types.WebCustomerAddress) error {
	address.Address = strings.TrimSpace(address.Address)
	address.Reference = strings.TrimSpace(address.Reference)
	// A district's ubigeo has six digits (150101); a province or department is shorter.
	if address.CityID < 10000 {
		return core.Err("Debe seleccionar el departamento, la provincia y el distrito.")
	}
	if len(address.Address) < 4 {
		return core.Err("La dirección debe tener al menos 4 caracteres.")
	}
	return nil
}

// saveWebCustomerAddresses writes addresses of one customer: new ones (ID <= 0) are inserted,
// existing ones must belong to the customer.
func saveWebCustomerAddresses(companyID, customerID int32, addresses []types.WebCustomerAddress) ([]types.WebCustomerAddress, error) {
	if len(addresses) == 0 {
		return addresses, nil
	}
	ownedIDs := map[int32]bool{}
	storedAddresses, err := listWebCustomerAddresses(companyID, customerID)
	if err != nil {
		return nil, err
	}
	for _, storedAddress := range storedAddresses {
		ownedIDs[storedAddress.ID] = true
	}

	newAddresses, changedAddresses := []types.WebCustomerAddress{}, []types.WebCustomerAddress{}
	for _, address := range addresses {
		if address.Status != 0 {
			if err := validateWebCustomerAddress(&address); err != nil {
				return nil, err
			}
		}
		address.CompanyID, address.CustomerID, address.Updated = companyID, customerID, core.SUnixTime()
		if address.ID <= 0 {
			address.ID, address.Status = 0, 1
			newAddresses = append(newAddresses, address)
		} else if ownedIDs[address.ID] {
			changedAddresses = append(changedAddresses, address)
		} else {
			return nil, core.Err("La dirección no pertenece al cliente.")
		}
	}
	if len(newAddresses) > 0 {
		if err := db.Insert(&newAddresses); err != nil {
			return nil, core.Err("Error al guardar la dirección:", err)
		}
	}
	if len(changedAddresses) > 0 {
		if err := db.UpdateExclude(&changedAddresses); err != nil {
			return nil, core.Err("Error al actualizar la dirección:", err)
		}
	}
	return append(newAddresses, changedAddresses...), nil
}

// loadWebStoreConfig reads the store's cached company config: flags, Culqi mode, sites, series.
func loadWebStoreConfig(companyID int32) (*config.CompanyConfig, error) {
	companyConfig, err := cloud.LoadCompanyConfig(companyID)
	if err != nil {
		return nil, core.Err("Error al leer la configuración de la tienda:", err)
	}
	return companyConfig, nil
}

// PostWebCustomerCode mails a 6-digit code: to register, to log in without a password, or to
// confirm an email at checkout.
func PostWebCustomerCode(req *core.HandlerArgs) core.HandlerResponse {
	body := struct{ Email string }{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	companyID := webStoreCompanyID(req)
	email := types.NormalizeCustomerEmail(body.Email)
	if companyID <= 0 || !webCustomerEmailPattern.MatchString(email) {
		return req.MakeErr("El correo electrónico no es válido.")
	}
	result, err := sendWebCustomerCode(req, companyID, email)
	if lockErr, isLockErr := err.(*core.LockError); isLockErr {
		return lockErr.Response(req)
	}
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(result)
}

// webCustomerSessionRequest is the body of the routes that open a session from an email: the
// code (when one was mailed), the profile typed so far and the addresses a guest saved in the
// browser before having an account.
type webCustomerSessionRequest struct {
	Email     string
	Code      string
	Profile   webCustomerProfile
	Addresses []types.WebCustomerAddress
	// PaymentMethod is read by the checkout identification only (types.WebPayment*).
	PaymentMethod int8
}

// PostWebCustomerVerify exchanges a mailed code for a session: it logs into the customer that owns
// the email, or creates it (confirmed) when there is none.
func PostWebCustomerVerify(req *core.HandlerArgs) core.HandlerResponse {
	body := webCustomerSessionRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	companyID := webStoreCompanyID(req)
	email := types.NormalizeCustomerEmail(body.Email)
	if companyID <= 0 || email == "" || body.Code == "" {
		return req.MakeErr("Faltan el correo o el código.")
	}
	if err := consumeWebCustomerCode(companyID, email, body.Code); err != nil {
		return req.MakeErr(err)
	}

	customer, err := findWebCustomerByEmail(companyID, email)
	if err != nil {
		return req.MakeErr(err)
	}
	if customer == nil {
		// Registering: the profile is required. A code-only login of an unknown email lands here
		// too, and is answered by asking the shopper to register.
		companyConfig, err := loadWebStoreConfig(companyID)
		if err != nil {
			return req.MakeErr(err)
		}
		if err := validateWebCustomerProfile(&body.Profile, companyConfig.Company.Flags); err != nil {
			return req.MakeErr("No existe una cuenta con este correo. Regístrese: ", err)
		}
		ipKey, _ := req.ClientIPKey()
		if customer, err = createWebCustomer(companyID, email, body.Profile, true, ipKey); err != nil {
			return req.MakeErr(err)
		}
	}
	if err := confirmAndFillWebCustomer(customer, body.Profile); err != nil {
		return req.MakeErr("Error al confirmar el correo:", err)
	}
	if _, err := saveWebCustomerAddresses(companyID, customer.ID, body.Addresses); err != nil {
		return req.MakeErr(err)
	}
	return makeWebCustomerSession(req, customer)
}

// PostWebCustomerLogin opens a session with email and password.
func PostWebCustomerLogin(req *core.HandlerArgs) core.HandlerResponse {
	body := struct{ Email, Password string }{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	companyID := webStoreCompanyID(req)
	customer, err := findWebCustomerByEmail(companyID, types.NormalizeCustomerEmail(body.Email))
	if err != nil {
		return req.MakeErr(err)
	}
	// One message for "no such customer", "no password set" and "wrong password", so the login
	// does not tell a stranger which emails have an account.
	if customer == nil || customer.Status == 0 || !customer.HasPassword() ||
		bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(body.Password)) != nil {
		return req.MakeErrCode("Correo o contraseña incorrectos. También puede ingresar con un código enviado a su correo.", 401)
	}
	return makeWebCustomerSession(req, customer)
}

// PostWebCheckoutIdentify runs when a guest presses pay: it decides whether the email needs a
// code before the order is placed (docs/WEB_CHECKOUT_PLAN.md, "Email confirmation at checkout").
//   - The email has a customer (confirmed or not): a code, always. Whoever types the email may
//     not be who created that customer, so their data never reaches it without the code.
//   - A new email, and the company asks for confirmation on this payment kind (flag 10 online,
//     flag 14 deferred): a code, then PostWebCustomerVerify creates the customer confirmed.
//   - A new email otherwise: the customer is created unconfirmed and the session returned now.
func PostWebCheckoutIdentify(req *core.HandlerArgs) core.HandlerResponse {
	body := webCustomerSessionRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	companyID := webStoreCompanyID(req)
	email := types.NormalizeCustomerEmail(body.Email)
	if companyID <= 0 || !webCustomerEmailPattern.MatchString(email) {
		return req.MakeErr("El correo electrónico no es válido.")
	}
	if body.PaymentMethod < types.WebPaymentCard || body.PaymentMethod > types.WebPaymentDeferred {
		return req.MakeErr("Debe elegir un medio de pago.")
	}
	companyConfig, err := loadWebStoreConfig(companyID)
	if err != nil {
		return req.MakeErr(err)
	}
	if err := validateWebCustomerProfile(&body.Profile, companyConfig.Company.Flags); err != nil {
		return req.MakeErr(err)
	}

	existingCustomer, err := findWebCustomerByEmail(companyID, email)
	if err != nil {
		return req.MakeErr(err)
	}
	confirmationFlag := core.If(types.IsOnlineWebPayment(body.PaymentMethod),
		types.CompanyFlagConfirmEmailOnOnlinePayment, types.CompanyFlagConfirmEmailOnDeferredPayment)

	if existingCustomer != nil || core.HasCompanyFlag(companyConfig.Company.Flags, confirmationFlag) {
		result, err := sendWebCustomerCode(req, companyID, email)
		if lockErr, isLockErr := err.(*core.LockError); isLockErr {
			return lockErr.Response(req)
		}
		if err != nil {
			return req.MakeErr(err)
		}
		return req.MakeResponse(map[string]any{"CodeRequired": true, "Sent": result.Sent,
			"RetryAfterSeconds": result.RetryAfterSeconds})
	}

	// No code: the customer is created unconfirmed. Bounded per IP, under the same lock as the
	// codes, so one connection cannot mint customers (and their pending orders) without end.
	ipKey, hasClientIP := req.ClientIPKey()
	if !hasClientIP {
		return req.MakeErr("No se pudo determinar el origen de la solicitud.")
	}
	ipLock, lockErr := core.AcquireLock(context.Background(), core.ActionWebCustomerByIP, ipKey, 2)
	if lockErr != nil {
		return lockErr.Response(req)
	}
	defer ipLock.Release()
	recentCustomers := []types.WebCustomer{}
	query := db.Query(&recentCustomers)
	query.Select(query.ID, query.Created).CompanyID.Equals(companyID).CreatedIP.Equals(ipKey)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al verificar el origen de la solicitud:", err)
	}
	windowStart := core.SUnixTime() - webCustomerWindowSUnits
	createdInWindow := 0
	for _, recentCustomer := range recentCustomers {
		if recentCustomer.Created >= windowStart {
			createdInWindow++
		}
	}
	if createdInWindow >= webCustomerMaxCreatedPerIP {
		return req.MakeErrCode("Se alcanzó el máximo de registros desde esta conexión. Intente en una hora.", 429)
	}

	customer, err := createWebCustomer(companyID, email, body.Profile, false, ipKey)
	if err != nil {
		return req.MakeErr(err)
	}
	if _, err := saveWebCustomerAddresses(companyID, customer.ID, body.Addresses); err != nil {
		return req.MakeErr(err)
	}
	return makeWebCustomerSession(req, customer)
}

// GetWebCustomer returns the logged-in customer's profile and addresses.
func GetWebCustomer(req *core.HandlerArgs) core.HandlerResponse {
	_, customer, err := requireWebCustomer(req)
	if err != nil {
		return req.MakeErrCode(err.Error(), 401)
	}
	return makeWebCustomerSession(req, customer)
}

// PostWebCustomer saves the customer's own profile (Mis Datos). The email is the account's identity
// and is not editable here.
func PostWebCustomer(req *core.HandlerArgs) core.HandlerResponse {
	companyID, customer, err := requireWebCustomer(req)
	if err != nil {
		return req.MakeErrCode(err.Error(), 401)
	}
	profile := webCustomerProfile{}
	if err := json.Unmarshal([]byte(*req.Body), &profile); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	companyConfig, err := loadWebStoreConfig(companyID)
	if err != nil {
		return req.MakeErr(err)
	}
	if err := validateWebCustomerProfile(&profile, companyConfig.Company.Flags); err != nil {
		return req.MakeErr(err)
	}
	customer.FirstName, customer.LastName = profile.FirstName, profile.LastName
	customer.Phone, customer.DocumentNumber = profile.Phone, profile.DocumentNumber
	customer.Updated = core.SUnixTime()
	customerTable := db.TableOf[types.WebCustomer]()
	if err := db.Update(&[]types.WebCustomer{*customer}, customerTable.FirstName, customerTable.LastName,
		customerTable.Phone, customerTable.DocumentNumber, customerTable.Updated); err != nil {
		return req.MakeErr("Error al guardar los datos:", err)
	}
	return req.MakeResponse(customer.PublicWebCustomer())
}

// PostWebCustomerPassword sets or changes the customer's password. Changing one needs the current
// password; setting the first one does not, since the session itself came from an emailed code.
func PostWebCustomerPassword(req *core.HandlerArgs) core.HandlerResponse {
	_, customer, err := requireWebCustomer(req)
	if err != nil {
		return req.MakeErrCode(err.Error(), 401)
	}
	body := struct{ CurrentPassword, NewPassword string }{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	if len(body.NewPassword) < webCustomerMinPasswordLength {
		return req.MakeErr(fmt.Sprintf("La contraseña debe tener al menos %v caracteres.", webCustomerMinPasswordLength))
	}
	if customer.HasPassword() &&
		bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(body.CurrentPassword)) != nil {
		return req.MakeErr("La contraseña actual no es correcta.")
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return req.MakeErr("Error al proteger la contraseña:", err)
	}
	customer.PasswordHash = string(passwordHash)
	customer.Updated = core.SUnixTime()
	customerTable := db.TableOf[types.WebCustomer]()
	if err := db.Update(&[]types.WebCustomer{*customer}, customerTable.PasswordHash, customerTable.Updated); err != nil {
		return req.MakeErr("Error al guardar la contraseña:", err)
	}
	return req.MakeResponse(map[string]any{"HasPassword": true})
}

// PostWebCustomerAddress creates, edits or deletes (Status 0) one of the customer's addresses.
func PostWebCustomerAddress(req *core.HandlerArgs) core.HandlerResponse {
	companyID, customer, err := requireWebCustomer(req)
	if err != nil {
		return req.MakeErrCode(err.Error(), 401)
	}
	address := types.WebCustomerAddress{}
	if err := json.Unmarshal([]byte(*req.Body), &address); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	saved, err := saveWebCustomerAddresses(companyID, customer.ID, []types.WebCustomerAddress{address})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(saved[0])
}
