package core

import "testing"

func TestHasAccesoNivelRequiresRequestedOrHigherLevel(t *testing.T) {
	// Store packed accesses exactly as auth keeps them: sorted by access ID and granted level.
	handlerArgs := HandlerArgs{
		Usuario:      &UsuarioToken{ID: 5},
		accesosNivel: []uint16{makeAccesoNivelUint16(8, 1), makeAccesoNivelUint16(12, 3)},
	}

	if handlerArgs.HasAccesoNivel(8, 2) {
		t.Fatalf("expected acceso 8 nivel 1 to be rejected when nivel 2 is required")
	}

	if !handlerArgs.HasAccesoNivel(8, 1) {
		t.Fatalf("expected acceso 8 nivel 1 to satisfy nivel 1")
	}
}

func TestHasAccesoNivelAcceptsHigherGrantedLevel(t *testing.T) {
	// Higher granted levels must continue to authorize lower requirements for the same access ID.
	handlerArgs := HandlerArgs{
		Usuario:      &UsuarioToken{ID: 5},
		accesosNivel: []uint16{makeAccesoNivelUint16(8, 3)},
	}

	if !handlerArgs.HasAccesoNivel(8, 2) {
		t.Fatalf("expected acceso 8 nivel 3 to satisfy nivel 2")
	}
}
