package core

import (
	"app/core/types"
	"app/db"
	"fmt"
	"strings"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

// SaasCompanyID owns the platform: emails that belong to no tenant (sign-up, contact form) are
// stored under it.
const SaasCompanyID = int32(1)

// EmailAttachment is one file sent with the email. Its MIME type comes from the name's extension.
type EmailAttachment struct {
	Name    string
	Content []byte
}

// EmailMessage is one HTML email. To needs at least one address; Cc and Bcc are optional.
type EmailMessage struct {
	To  []string
	Cc  []string
	Bcc []string
	// FromName is the display name of the sender (a tenant's name); the address is always the
	// platform's SMTP account. Empty = "Genix".
	FromName string
	// ReplyTo is where the recipient's answer goes (the seller), since the From address is shared.
	ReplyTo     string
	Subject     string
	HTMLBody    string
	Attachments []EmailAttachment
}

// SendEmail delivers one HTML message through the SMTP server configured in config.toml, and
// records it in the sent_emails table of companyID (the Email Inbox page). With [smtp].store_only
// nothing goes out over SMTP: storing it is the send, so dev/QAS can exercise every email flow
// without mailing anybody.
//
// A fresh connection per message is deliberate: the backend sends mail rarely (sign-up
// verification today) and runs in Lambda, where a pooled client would be torn down between
// invocations anyway.
func SendEmail(companyID int32, message EmailMessage) error {
	if len(message.To) == 0 {
		return Err("No se indicó el destinatario del correo.")
	}
	if Env.SMTP_STORE_ONLY {
		Log("SendEmail:: [smtp].store_only, el correo a", message.To, "solo se registra en sent_emails")
		return storeSentEmail(companyID, message, types.SentEmailStatusStored, "")
	}
	if Env.SMTP_PORT == 0 || len(Env.SMTP_HOST) == 0 ||
		len(Env.SMTP_EMAIL) == 0 || len(Env.SMTP_PASSWORD) == 0 {
		return Err("El envío de correo no está configurado: faltan datos SMTP en config.toml.")
	}

	if sendErr := sendEmailBySMTP(message); sendErr != nil {
		// The failure is stored too, so the inbox shows what was attempted and why it did not leave.
		_ = storeSentEmail(companyID, message, types.SentEmailStatusFailed, sendErr.Error())
		return sendErr
	}
	Log("SendEmail:: correo enviado a", message.To, "|", message.Subject)
	// The email already left: a failed store is only logged, or the caller would retry and send it twice.
	_ = storeSentEmail(companyID, message, types.SentEmailStatusSent, "")
	return nil
}

func sendEmailBySMTP(message EmailMessage) error {
	server := mail.NewSMTPClient()
	server.Host = Env.SMTP_HOST
	server.Port = int(Env.SMTP_PORT)
	server.Username = Env.SMTP_USER
	server.Password = Env.SMTP_PASSWORD
	server.Encryption = mail.EncryptionSTARTTLS
	// The library defaults to 10s + 10s, which lets one hung mail server stall a caller for ~20s.
	// Sign-up sends this while holding the per-IP lock, so the bound here is what keeps the
	// critical section under the lock's lease; raising these means raising that lease too.
	server.ConnectTimeout = 4 * time.Second
	server.SendTimeout = 6 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		return Err("Error al conectar con el servidor SMTP:", err)
	}

	email := mail.NewMSG()
	email.SetFrom(senderAddress(message))
	if message.ReplyTo != "" {
		email.SetReplyTo(message.ReplyTo)
	}
	email.AddTo(message.To...).
		AddCc(message.Cc...).
		AddBcc(message.Bcc...).
		SetSubject(message.Subject).
		SetBody(mail.TextHTML, message.HTMLBody)
	for _, attachment := range message.Attachments {
		email.Attach(&mail.File{Name: attachment.Name, Data: attachment.Content})
	}

	if email.Error != nil {
		return Err("Error al componer el correo:", email.Error)
	}
	if err := email.Send(smtpClient); err != nil {
		return Err("Error al enviar el correo:", err)
	}
	return nil
}

// senderAddress is "<FromName> <platform address>". Quotes and angle brackets are dropped from the
// name so a tenant's name cannot break the header.
func senderAddress(message EmailMessage) string {
	fromName := strings.Map(func(char rune) rune {
		if strings.ContainsRune("\"<>\r\n", char) {
			return -1
		}
		return char
	}, strings.TrimSpace(message.FromName))
	if fromName == "" {
		fromName = "Genix"
	}
	return fmt.Sprintf("%v <%v>", fromName, Env.SMTP_EMAIL)
}

// storeSentEmail writes the inbox row of an email, with only the names of its attachments.
func storeSentEmail(companyID int32, message EmailMessage, status int8, sendError string) error {
	sentEmail := types.SentEmail{
		CompanyID: companyID, To: message.To, Cc: message.Cc, Bcc: message.Bcc,
		Subject: message.Subject, HTMLBody: message.HTMLBody, Sender: senderAddress(message), ReplyTo: message.ReplyTo,
		SendError: sendError, Status: status,
	}
	for _, attachment := range message.Attachments {
		sentEmail.AttachmentNames = append(sentEmail.AttachmentNames, attachment.Name)
	}
	if err := db.Insert(&[]types.SentEmail{sentEmail}); err != nil {
		return Err("SendEmail:: no se pudo registrar el correo a", message.To, "en sent_emails:", err)
	}
	return nil
}
