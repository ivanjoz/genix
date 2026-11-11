package types

import (
	"strings"
	"testing"
)

func TestNormalizeEmailTemplateRefusesUnknownPlaceholders(t *testing.T) {
	template := CrmEmailTemplate{Name: " Seguimiento ", Subject: "Hola {{ client.name }}", BodyHTML: "<p>{{opportunity.title}} por {{quote.total}}</p>"}
	if err := NormalizeEmailTemplate(&template); err != nil || template.Name != "Seguimiento" {
		t.Fatalf("a known placeholder, spaces inside braces, and a quote one are fine: %v", err)
	}
	for caseName, broken := range map[string]CrmEmailTemplate{
		"typo in the body":    {Name: "A", Subject: "S", BodyHTML: "{{client.nmae}}"},
		"typo in the subject": {Name: "A", Subject: "{{owner}}", BodyHTML: "B"},
		"no name":             {Name: " ", Subject: "S", BodyHTML: "B"},
		"no body":             {Name: "A", Subject: "S", BodyHTML: ""},
	} {
		if err := NormalizeEmailTemplate(&broken); err == nil {
			t.Errorf("%s: expected an error", caseName)
		}
	}
}

func TestRenderEmailTemplateEscapesTheBodyOnly(t *testing.T) {
	template := CrmEmailTemplate{Name: "T", Subject: "Para {{client.name}}", BodyHTML: "<p>{{client.name}} · {{opportunity.amount}} · {{owner.name}}</p>"}
	subject, body, err := RenderEmailTemplate(template, EmailRenderContext{ClientName: "A & <B>", OpportunityAmount: EmailAmount(123456789), OwnerName: "Ana"})
	if err != nil {
		t.Fatal(err)
	}
	if subject != "Para A & <B>" {
		t.Fatalf("subject is plain text: %q", subject)
	}
	if body != "<p>A &amp; &lt;B&gt; · 1,234,567.89 · Ana</p>" {
		t.Fatalf("body escapes values: %q", body)
	}
}

func TestRenderEmailTemplateDropsTheEditorWrapper(t *testing.T) {
	template := CrmEmailTemplate{Name: "T", Subject: "S", BodyHTML: `<p>Hola <span data-variable="client.name">{{client.name}}</span>, <span style="color: red">ok</span></p>`}
	_, body, err := RenderEmailTemplate(template, EmailRenderContext{ClientName: "Ana & Co"})
	if err != nil {
		t.Fatal(err)
	}
	if body != `<p>Hola Ana &amp; Co, <span style="color: red">ok</span></p>` {
		t.Fatalf("the variable span is dropped, other spans stay: %q", body)
	}
}

func TestRenderEmailTemplateNeedsAQuoteForQuotePlaceholders(t *testing.T) {
	template := CrmEmailTemplate{Name: "Cotización", Subject: "Cotización {{quote.number}}", BodyHTML: "Total {{quote.total}}"}
	if _, _, err := RenderEmailTemplate(template, EmailRenderContext{}); err == nil || !strings.Contains(err.Error(), "cotización") {
		t.Fatalf("a quote placeholder without a quote is an error: %v", err)
	}
	subject, body, err := RenderEmailTemplate(template, EmailRenderContext{Quote: &EmailQuoteContext{Number: "COT-1", Total: "10.00"}})
	if err != nil || subject != "Cotización COT-1" || body != "Total 10.00" {
		t.Fatalf("with a quote: %q %q %v", subject, body, err)
	}
}

func TestEmailAmount(t *testing.T) {
	for amount, expected := range map[int32]string{0: "0.00", 5: "0.05", 100000: "1,000.00", -250050: "-2,500.50", 99999999: "999,999.99"} {
		if formatted := EmailAmount(amount); formatted != expected {
			t.Errorf("%d: %q, expected %q", amount, formatted, expected)
		}
	}
}

func TestNormalizeOpportunityEmail(t *testing.T) {
	payload := CrmOpportunityEmailPayload{To: []string{" ana@x.pe ", ""}, Cc: []string{"b@x.pe"}, Subject: " Hola ", HTMLBody: "<p>Hola</p>"}
	if err := NormalizeOpportunityEmail(&payload); err != nil || len(payload.To) != 1 || payload.To[0] != "ana@x.pe" || payload.Subject != "Hola" {
		t.Fatalf("trims and drops blanks: %+v %v", payload, err)
	}
	for caseName, broken := range map[string]CrmOpportunityEmailPayload{
		"no recipient":    {Subject: "S", HTMLBody: "B"},
		"bad address":     {To: []string{"Ana <ana@x.pe>"}, Subject: "S", HTMLBody: "B"},
		"no subject":      {To: []string{"a@x.pe"}, HTMLBody: "B"},
		"raw placeholder": {To: []string{"a@x.pe"}, Subject: "S", HTMLBody: "Hola {{client.name}}"},
		"too many people": {To: strings.Split(strings.Repeat("a@x.pe,", 21), ",")[:21], Subject: "S", HTMLBody: "B"},
	} {
		if err := NormalizeOpportunityEmail(&broken); err == nil {
			t.Errorf("%s: expected an error", caseName)
		}
	}
	text := EmailActivityText(CrmOpportunityEmailPayload{Subject: "Hola", To: []string{"a@x.pe"}, Cc: []string{"b@x.pe"}})
	if text != "Hola\nPara: a@x.pe\nCC: b@x.pe" {
		t.Fatalf("activity text: %q", text)
	}
}
