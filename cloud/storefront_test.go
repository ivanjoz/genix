package main

import "testing"

func TestUpsertTomlSection(t *testing.T) {
	values := [][2]string{{"hosted_zone_id", "Z1"}, {"bucket", "b1"}}

	t.Run("crea la sección al final", func(t *testing.T) {
		got := upsertTomlSection("[aws]\nregion = \"us-east-1\"\n", "storefront", values)
		want := "[aws]\nregion = \"us-east-1\"\n\n[storefront]\nhosted_zone_id = \"Z1\"\nbucket = \"b1\"\n"
		if got != want {
			t.Fatalf("got:\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("reemplaza dentro de la sección y conserva comentarios y otras secciones", func(t *testing.T) {
		input := "[storefront]\nbucket = \"viejo\" # comentario\n\n[otra]\nbucket = \"no-tocar\"\n"
		got := upsertTomlSection(input, "storefront", values)
		want := "[storefront]\nhosted_zone_id = \"Z1\"\nbucket = \"b1\" # comentario\n\n[otra]\nbucket = \"no-tocar\"\n"
		if got != want {
			t.Fatalf("got:\n%q\nwant:\n%q", got, want)
		}
	})
}
