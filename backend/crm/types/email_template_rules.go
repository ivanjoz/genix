package types

import (
	"errors"
	"fmt"
	"html"
	"net/mail"
	"regexp"
	"strings"
)

const (
	maxEmailTemplateNameLength = 120
	maxEmailSubjectLength      = 200
	maxEmailBodyLength         = 200_000
	maxEmailRecipients         = 20
)

// EmailPlaceholders are the {{names}} a template may use, in the order the editor lists them.
// The quote ones only render when the email is sent with a quote (sales quotes, plan §7).
// Mirrored by EMAIL_PLACEHOLDERS in frontend/services/crm/crm-email-template.ts.
var EmailPlaceholders = []string{
	"client.name", "client.email", "opportunity.title", "opportunity.amount", "owner.name", "owner.email",
	"company.name", "quote.number", "quote.total", "quote.valid_until",
}

var emailPlaceholderPattern = regexp.MustCompile(`\{\{\s*([a-z_.]+)\s*\}\}`)

// EmailRenderContext holds the values of the placeholders for one email. Quote is nil when the
// email is not about a quote.
type EmailRenderContext struct {
	ClientName       string
	ClientEmail      string
	OpportunityTitle string
	// OpportunityAmount is already formatted for the reader (EmailAmount).
	OpportunityAmount string
	OwnerName         string
	OwnerEmail        string
	CompanyName       string
	Quote             *EmailQuoteContext
}

type EmailQuoteContext struct {
	Number     string
	Total      string
	ValidUntil string
}

// usedPlaceholders lists the placeholder names a text uses, in order of appearance.
func usedPlaceholders(text string) []string {
	names := []string{}
	for _, match := range emailPlaceholderPattern.FindAllStringSubmatch(text, -1) {
		names = append(names, match[1])
	}
	return names
}

func isKnownPlaceholder(name string) bool {
	for _, known := range EmailPlaceholders {
		if known == name {
			return true
		}
	}
	return false
}

// NormalizeEmailTemplate trims and checks a template before it is saved. An unknown placeholder is
// refused here, so a sent email never carries a silent blank or a raw {{typo}}.
func NormalizeEmailTemplate(template *CrmEmailTemplate) error {
	template.Name = strings.TrimSpace(template.Name)
	template.Subject = strings.TrimSpace(template.Subject)
	template.BodyHTML = strings.TrimSpace(template.BodyHTML)
	if template.Name == "" || len(template.Name) > maxEmailTemplateNameLength {
		return fmt.Errorf("La plantilla necesita un nombre de hasta %d caracteres.", maxEmailTemplateNameLength)
	}
	if template.Subject == "" || len(template.Subject) > maxEmailSubjectLength {
		return fmt.Errorf("La plantilla necesita un asunto de hasta %d caracteres.", maxEmailSubjectLength)
	}
	if template.BodyHTML == "" || len(template.BodyHTML) > maxEmailBodyLength {
		return errors.New("La plantilla necesita un cuerpo, de hasta 200 KB.")
	}
	for _, name := range usedPlaceholders(template.Subject + "\n" + template.BodyHTML) {
		if !isKnownPlaceholder(name) {
			return fmt.Errorf("La variable {{%s}} no existe. Use: %s.", name, strings.Join(EmailPlaceholders, ", "))
		}
	}
	return nil
}

// RenderEmailTemplate fills the placeholders of a template for one email. The values are escaped
// in the HTML body (a client named "<b>" stays text) and left as they are in the subject, which is
// plain text. A quote placeholder without a quote is an error: the template is for quote emails.
func RenderEmailTemplate(template CrmEmailTemplate, context EmailRenderContext) (subject string, body string, err error) {
	values := map[string]string{
		"client.name": context.ClientName, "client.email": context.ClientEmail,
		"opportunity.title": context.OpportunityTitle, "opportunity.amount": context.OpportunityAmount,
		"owner.name": context.OwnerName, "owner.email": context.OwnerEmail, "company.name": context.CompanyName,
	}
	if context.Quote != nil {
		values["quote.number"], values["quote.total"], values["quote.valid_until"] =
			context.Quote.Number, context.Quote.Total, context.Quote.ValidUntil
	}
	var renderError error
	fill := func(text string, escape bool) string {
		return emailPlaceholderPattern.ReplaceAllStringFunc(text, func(placeholder string) string {
			name := emailPlaceholderPattern.FindStringSubmatch(placeholder)[1]
			value, isAvailable := values[name]
			if !isAvailable {
				if renderError == nil {
					if strings.HasPrefix(name, "quote.") {
						renderError = fmt.Errorf("La plantilla %q usa {{%s}}: solo se envía con una cotización.", template.Name, name)
					} else {
						renderError = fmt.Errorf("La variable {{%s}} no existe.", name)
					}
				}
				return placeholder
			}
			if escape {
				return html.EscapeString(value)
			}
			return value
		})
	}
	subject, body = fill(template.Subject, false), fill(template.BodyHTML, true)
	return subject, body, renderError
}

// EmailAmount formats an amount (the cents of the company currency) as 1,234.56.
func EmailAmount(amount int32) string {
	sign := ""
	if amount < 0 {
		sign, amount = "-", -amount
	}
	whole := fmt.Sprintf("%d", amount/100)
	grouped := []string{}
	for len(whole) > 3 {
		grouped = append([]string{whole[len(whole)-3:]}, grouped...)
		whole = whole[:len(whole)-3]
	}
	grouped = append([]string{whole}, grouped...)
	return fmt.Sprintf("%s%s.%02d", sign, strings.Join(grouped, ","), amount%100)
}

// NormalizeOpportunityEmail checks an email a seller is about to send: real addresses, at most
// maxEmailRecipients in all, a subject and a body. Addresses are trimmed and blank ones dropped.
func NormalizeOpportunityEmail(payload *CrmOpportunityEmailPayload) error {
	var err error
	if payload.To, err = normalizeEmailAddresses(payload.To); err != nil {
		return err
	}
	if payload.Cc, err = normalizeEmailAddresses(payload.Cc); err != nil {
		return err
	}
	if len(payload.To) == 0 {
		return errors.New("Indique al menos un destinatario.")
	}
	if len(payload.To)+len(payload.Cc) > maxEmailRecipients {
		return fmt.Errorf("Un correo puede tener hasta %d destinatarios.", maxEmailRecipients)
	}
	payload.Subject = strings.TrimSpace(payload.Subject)
	if payload.Subject == "" || len(payload.Subject) > maxEmailSubjectLength {
		return fmt.Errorf("El correo necesita un asunto de hasta %d caracteres.", maxEmailSubjectLength)
	}
	// A rendered placeholder left in the text means the seller typed one by hand: it would go out raw.
	if placeholders := usedPlaceholders(payload.Subject + payload.HTMLBody); len(placeholders) > 0 {
		return fmt.Errorf("El correo todavía contiene {{%s}}: reemplácelo antes de enviar.", placeholders[0])
	}
	payload.HTMLBody = strings.TrimSpace(payload.HTMLBody)
	if payload.HTMLBody == "" || len(payload.HTMLBody) > maxEmailBodyLength {
		return errors.New("El correo necesita un cuerpo, de hasta 200 KB.")
	}
	return nil
}

func normalizeEmailAddresses(addresses []string) ([]string, error) {
	normalized := []string{}
	for _, address := range addresses {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}
		parsed, err := mail.ParseAddress(address)
		if err != nil || parsed.Address != address {
			return nil, fmt.Errorf("La dirección %q no es un correo válido.", address)
		}
		normalized = append(normalized, address)
	}
	return normalized, nil
}

// EmailActivityText is what the opportunity's timeline shows for a sent email.
func EmailActivityText(payload CrmOpportunityEmailPayload) string {
	text := fmt.Sprintf("%s\nPara: %s", payload.Subject, strings.Join(payload.To, ", "))
	if len(payload.Cc) > 0 {
		text += "\nCC: " + strings.Join(payload.Cc, ", ")
	}
	return text
}
