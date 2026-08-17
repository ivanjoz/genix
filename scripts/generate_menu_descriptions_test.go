package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const documentationFrontmatter = `---
schema: 1
page_id: system.companies
route: /system/companies
title: Companies (Empresas)
status: implemented
visibility: saas
description_en: >-
  Tenant company management, SaaS only. Create and edit companies: name and RUC.
description_es: >-
  Gestión de empresas (tenants), exclusivo SaaS.
---

# Companies (Empresas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Prose that belongs to the RAG index only.
`

// La descripción viene del frontmatter y no debe arrastrar el marcador DOC-ID que la sigue.
func TestFrontmatterDescriptionsStopAtTheFrontmatter(t *testing.T) {
	spanish, english, err := parseFrontmatterDescriptions(documentationFrontmatter)
	if err != nil {
		t.Fatalf("frontmatter válido rechazado: %v", err)
	}

	wantEnglish := "Tenant company management, SaaS only. Create and edit companies: name and RUC."
	if english != wantEnglish {
		t.Fatalf("descripción EN incorrecta:\n got: %q\nwant: %q", english, wantEnglish)
	}
	if spanish != "Gestión de empresas (tenants), exclusivo SaaS." {
		t.Fatalf("descripción ES incorrecta: %q", spanish)
	}
	if strings.Contains(english, "DOC-ID") || strings.Contains(spanish, "DOC-ID") {
		t.Fatal("el marcador DOC-ID se filtró en la descripción del menú")
	}
}

// Los archivos sin frontmatter siguen usando los bloques "## DESCRIPTION::".
func TestLegacyStubStillParses(t *testing.T) {
	spanish, english, err := parseFrontmatterDescriptions("## DESCRIPTION::ES\nTexto.\n")
	if err != nil {
		t.Fatalf("stub sin frontmatter rechazado: %v", err)
	}
	if spanish != "" || english != "" {
		t.Fatalf("un stub sin frontmatter no debe producir descripciones: %q / %q", spanish, english)
	}

	descriptions := parseDescriptionBlocks("## DESCRIPTION::ES\nTexto.\n\n## DESCRIPTION::EN\nText.\n")
	if descriptions["ES"] != "Texto." || descriptions["EN"] != "Text." {
		t.Fatalf("bloques heredados incorrectos: %#v", descriptions)
	}
}

// Un stub olvidado tras migrar al frontmatter debe fallar en lugar de sobrescribir la descripción.
func TestDuplicateRouteFails(t *testing.T) {
	routesDir := filepath.Join(t.TempDir(), "routes")
	routeDir := filepath.Join(routesDir, "system", "companies")
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		t.Fatalf("crear ruta de prueba: %v", err)
	}
	writeFile := func(name, content string) {
		if err := os.WriteFile(filepath.Join(routeDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("escribir %s: %v", name, err)
		}
	}
	writeFile("+page.svelte", "<div></div>")
	writeFile("DOCUMENTATION.md", documentationFrontmatter)

	if _, err := collectMenuDescriptions(routesDir); err != nil {
		t.Fatalf("una sola descripción por ruta debe pasar: %v", err)
	}

	writeFile("empresas.md", "## DESCRIPTION::ES\nTexto.\n\n## DESCRIPTION::EN\nText.\n")

	_, err := collectMenuDescriptions(routesDir)
	if err == nil {
		t.Fatal("dos archivos para la misma ruta deben fallar")
	}
	if !strings.Contains(err.Error(), "/system/companies") {
		t.Fatalf("el error debe nombrar la ruta duplicada: %v", err)
	}
}
