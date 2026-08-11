package core

import "testing"

func TestRAGConfigDefaults(t *testing.T) {
	configuredFile := fileConfig{}
	configuredEnv := EnvStruct{}

	configuredFile.applyToEnv(&configuredEnv)

	if configuredEnv.QDRANT_HOST != "127.0.0.1" {
		t.Fatalf("QDRANT_HOST = %q, want loopback for a private default", configuredEnv.QDRANT_HOST)
	}
	if configuredEnv.QDRANT_GRPC_PORT != defaultQdrantGRPCPort {
		t.Fatalf("QDRANT_GRPC_PORT = %d, want %d", configuredEnv.QDRANT_GRPC_PORT, defaultQdrantGRPCPort)
	}
	if configuredEnv.QDRANT_COLLECTION != defaultQdrantCollection {
		t.Fatalf("QDRANT_COLLECTION = %q, want %q", configuredEnv.QDRANT_COLLECTION, defaultQdrantCollection)
	}
	if configuredEnv.EMBEDDING_PROVIDER != defaultEmbeddingProvider {
		t.Fatalf("EMBEDDING_PROVIDER = %q, want %q", configuredEnv.EMBEDDING_PROVIDER, defaultEmbeddingProvider)
	}
	if configuredEnv.EMBEDDING_DIMENSIONS != defaultEmbeddingDimensions {
		t.Fatalf("EMBEDDING_DIMENSIONS = %d, want %d", configuredEnv.EMBEDDING_DIMENSIONS, defaultEmbeddingDimensions)
	}
}

func TestRAGConfigKeepsPublicHostExplicit(t *testing.T) {
	configuredFile := fileConfig{}
	configuredFile.Qdrant.Public = true
	configuredEnv := EnvStruct{}

	configuredFile.applyToEnv(&configuredEnv)

	// A remote backend must not silently dial its own loopback when Qdrant is public.
	if configuredEnv.QDRANT_HOST != "" {
		t.Fatalf("QDRANT_HOST = %q, want blank so subsystem validation reports the missing host", configuredEnv.QDRANT_HOST)
	}
}
