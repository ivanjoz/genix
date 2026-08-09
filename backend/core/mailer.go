package core

import (
	"fmt"

	mail "github.com/xhit/go-simple-mail/v2"
)

// SendEmail delivers one HTML message through the SMTP server configured in config.toml.
//
// A fresh connection per message is deliberate: the backend sends mail rarely (sign-up
// verification today) and runs in Lambda, where a pooled client would be torn down between
// invocations anyway.
func SendEmail(toAddress string, subject string, htmlBody string) error {
	if Env.SMTP_PORT == 0 || len(Env.SMTP_HOST) == 0 ||
		len(Env.SMTP_EMAIL) == 0 || len(Env.SMTP_PASSWORD) == 0 {
		return Err("El envío de correo no está configurado: faltan datos SMTP en config.toml.")
	}
	if len(toAddress) == 0 {
		return Err("No se indicó el destinatario del correo.")
	}

	server := mail.NewSMTPClient()
	server.Host = Env.SMTP_HOST
	server.Port = int(Env.SMTP_PORT)
	server.Username = Env.SMTP_USER
	server.Password = Env.SMTP_PASSWORD
	server.Encryption = mail.EncryptionSTARTTLS

	smtpClient, err := server.Connect()
	if err != nil {
		return Err("Error al conectar con el servidor SMTP:", err)
	}

	email := mail.NewMSG()
	email.SetFrom(fmt.Sprintf("Genix <%v>", Env.SMTP_EMAIL)).
		AddTo(toAddress).
		SetSubject(subject).
		SetBody(mail.TextHTML, htmlBody)

	if email.Error != nil {
		return Err("Error al componer el correo:", email.Error)
	}

	if err := email.Send(smtpClient); err != nil {
		return Err("Error al enviar el correo:", err)
	}

	Log("SendEmail:: correo enviado a", toAddress, "|", subject)
	return nil
}
