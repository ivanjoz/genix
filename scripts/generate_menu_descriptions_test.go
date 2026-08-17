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

// DOCUMENTATION.md es la única fuente: los stubs heredados ya no aportan descripciones.
func TestLegacyStubIsIgnored(t *testing.T) {
	routesDir := newTestRoute(t, map[string]string{
		"+page.svelte": "<div></div>",
		"usuarios.md":  "## DESCRIPTION::ES\nTexto.\n\n## DESCRIPTION::EN\nText.\n",
	})

	menuDescriptions, err := collectMenuDescriptions(routesDir)
	if err != nil {
		t.Fatalf("un stub heredado debe ignorarse, no fallar: %v", err)
	}
	if len(menuDescriptions) != 0 {
		t.Fatalf("el stub heredado no debe producir entradas: %#v", menuDescriptions)
	}
}

// Solo se indexan rutas reales: un DOCUMENTATION.md sin +page.svelte publicaría una ruta inexistente.
func TestDocumentationWithoutPageFails(t *testing.T) {
	routesDir := newTestRoute(t, map[string]string{
		documentationFileName: documentationFrontmatter,
	})

	_, err := collectMenuDescriptions(routesDir)
	if err == nil {
		t.Fatal("un documento sin +page.svelte debe fallar")
	}
	if !strings.Contains(err.Error(), "+page.svelte") {
		t.Fatalf("el error debe nombrar el archivo faltante: %v", err)
	}
}

// La ruta sale del directorio del documento, no del nombre del archivo.
func TestRouteComesFromTheDirectory(t *testing.T) {
	routesDir := newTestRoute(t, map[string]string{
		"+page.svelte":        "<div></div>",
		documentationFileName: documentationFrontmatter,
	})

	menuDescriptions, err := collectMenuDescriptions(routesDir)
	if err != nil {
		t.Fatalf("documento válido rechazado: %v", err)
	}
	if len(menuDescriptions) != 1 || menuDescriptions[0].Route != "/system/companies" {
		t.Fatalf("ruta incorrecta: %#v", menuDescriptions)
	}
}

// newTestRoute crea frontend/routes/system/companies con los archivos indicados.
func newTestRoute(t *testing.T, files map[string]string) string {
	t.Helper()
	routesDir := filepath.Join(t.TempDir(), "routes")
	routeDir := filepath.Join(routesDir, "system", "companies")
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		t.Fatalf("crear ruta de prueba: %v", err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(routeDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("escribir %s: %v", name, err)
		}
	}
	return routesDir
}
