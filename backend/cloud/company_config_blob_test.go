package cloud

import (
	config "app/config/types"
	"app/core"
	"bytes"
	"strings"
	"testing"
	"time"
)

// withTestSecret sets the project secret the object name and the cipher key both
// derive from, and restores whatever was there. core.Env is nil until the process
// reads its configuration, which a unit test does not do.
func withTestSecret(t *testing.T, secret string) {
	t.Helper()
	if core.Env == nil {
		core.Env = &core.EnvStruct{}
	}
	previous := core.Env.SECRET_PHRASE
	core.Env.SECRET_PHRASE = secret
	t.Cleanup(func() { core.Env.SECRET_PHRASE = previous })
}

func TestCompanyConfigObjectNameIsStableAndCompanyScoped(t *testing.T) {
	withTestSecret(t, "una-frase-secreta-de-prueba")

	name := companyConfigObjectName(42)
	if name != companyConfigObjectName(42) {
		t.Fatal("the same company produced two different names")
	}
	if !strings.HasPrefix(name, "c42_cf_") || !strings.HasSuffix(name, ".cb") {
		t.Fatalf("unexpected name shape: %v", name)
	}
	// 64 bits base64url without padding is 11 characters.
	tag := strings.TrimSuffix(strings.TrimPrefix(name, "c42_cf_"), ".cb")
	if len(tag) != 11 {
		t.Fatalf("expected an 11-character tag, got %q", tag)
	}
	if name == companyConfigObjectName(43) {
		t.Fatal("two companies share an object name")
	}
}

// The tag is a MAC, not a checksum: without the secret it must not be reproducible.
func TestCompanyConfigObjectNameDependsOnTheSecret(t *testing.T) {
	withTestSecret(t, "primera-frase-secreta")
	first := companyConfigObjectName(7)

	withTestSecret(t, "segunda-frase-secreta")
	if second := companyConfigObjectName(7); second == first {
		t.Fatal("the object name did not change with the secret")
	}
}

func TestSealAndOpenCompanyConfigRoundTrip(t *testing.T) {
	withTestSecret(t, "una-frase-secreta-de-prueba")
	payload := []byte("la configuración de la empresa")
	objectName := companyConfigObjectName(42)

	sealed, err := sealCompanyConfig(payload, objectName)
	if err != nil {
		t.Fatalf("sealCompanyConfig: %v", err)
	}
	if bytes.Contains(sealed, payload) {
		t.Fatal("the plaintext survived into the sealed blob")
	}

	opened, err := openCompanyConfig(sealed, objectName)
	if err != nil {
		t.Fatalf("openCompanyConfig: %v", err)
	}
	if !bytes.Equal(opened, payload) {
		t.Fatalf("round trip changed the payload: %q", opened)
	}
}

// The nonce is random, so two seals of the same payload must differ — otherwise a
// watcher of the bucket learns when a config did not change.
func TestSealCompanyConfigUsesAFreshNonce(t *testing.T) {
	withTestSecret(t, "una-frase-secreta-de-prueba")
	objectName := companyConfigObjectName(42)

	first, err := sealCompanyConfig([]byte("misma carga"), objectName)
	if err != nil {
		t.Fatalf("sealCompanyConfig: %v", err)
	}
	second, err := sealCompanyConfig([]byte("misma carga"), objectName)
	if err != nil {
		t.Fatalf("sealCompanyConfig: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("two seals of the same payload were identical")
	}
}

func TestOpenCompanyConfigRejectsTampering(t *testing.T) {
	withTestSecret(t, "una-frase-secreta-de-prueba")
	objectName := companyConfigObjectName(42)

	sealed, err := sealCompanyConfig([]byte("la configuración"), objectName)
	if err != nil {
		t.Fatalf("sealCompanyConfig: %v", err)
	}
	sealed[len(sealed)-1] ^= 0xFF

	if _, err := openCompanyConfig(sealed, objectName); err == nil {
		t.Fatal("a modified blob was accepted")
	}
}

// The object name is the AAD, so a blob moved onto another company's key must not
// open — that is what stops a copied object from becoming somebody else's config.
func TestOpenCompanyConfigRejectsAnotherCompanysName(t *testing.T) {
	withTestSecret(t, "una-frase-secreta-de-prueba")

	sealed, err := sealCompanyConfig([]byte("la configuración"), companyConfigObjectName(42))
	if err != nil {
		t.Fatalf("sealCompanyConfig: %v", err)
	}
	if _, err := openCompanyConfig(sealed, companyConfigObjectName(43)); err == nil {
		t.Fatal("a blob opened under a different company's object name")
	}
}

func TestOpenCompanyConfigRejectsATruncatedBlob(t *testing.T) {
	withTestSecret(t, "una-frase-secreta-de-prueba")
	if _, err := openCompanyConfig([]byte{1, 2, 3}, "c1_cf_x.cb"); err == nil {
		t.Fatal("a truncated blob was accepted")
	}
}

func TestCompanyConfigCipherKeyRequiresASecret(t *testing.T) {
	withTestSecret(t, "")
	if _, err := companyConfigCipherKey(); err == nil {
		t.Fatal("a key was derived without a configured secret")
	}
}

// withEmptyConfigCache isolates a test from whatever else touched the cache. The map
// is package state, so it outlives a single test function.
func withEmptyConfigCache(t *testing.T) {
	t.Helper()
	companyConfigCacheMutex.Lock()
	companyConfigCache = map[int32]companyConfigCacheEntry{}
	companyConfigCacheMutex.Unlock()
	t.Cleanup(func() {
		companyConfigCacheMutex.Lock()
		companyConfigCache = map[int32]companyConfigCacheEntry{}
		companyConfigCacheMutex.Unlock()
	})
}

func TestCompanyConfigCacheServesWhatWasWritten(t *testing.T) {
	withEmptyConfigCache(t)
	stored := &config.CompanyConfig{CompanyID: 42, Company: config.CompanyConfigCompany{RUC: "20000000001"}}

	writeCompanyConfigCache(42, stored)

	cached := readCompanyConfigCache(42)
	if cached == nil {
		t.Fatal("the entry just written was not served")
	}
	if cached != stored {
		t.Fatal("the cache served a different value than the one written")
	}
	if readCompanyConfigCache(43) != nil {
		t.Fatal("another company read this company's entry")
	}
}

// The expiry is checked on the read, so an entry past its TTL is never served even
// if nothing has swept it yet.
func TestCompanyConfigCacheDoesNotServeAnExpiredEntry(t *testing.T) {
	withEmptyConfigCache(t)
	companyConfigCache[42] = companyConfigCacheEntry{
		companyConfig: &config.CompanyConfig{CompanyID: 42},
		expiresAtUnix: time.Now().Add(-time.Second).UnixNano(),
	}

	if readCompanyConfigCache(42) != nil {
		t.Fatal("an expired entry was served")
	}
}

// The point of the sweep is memory: an expired entry holds a private key and a
// certificate, and nothing else in the process is going to drop it. The tick itself
// is not exercised — waiting twenty seconds for a three-line goroutine is not a test.
func TestSweepCompanyConfigCacheDropsOnlyExpiredEntries(t *testing.T) {
	withEmptyConfigCache(t)
	companyConfigCache[7] = companyConfigCacheEntry{
		companyConfig: &config.CompanyConfig{CompanyID: 7},
		expiresAtUnix: time.Now().Add(-time.Second).UnixNano(),
	}
	companyConfigCache[8] = companyConfigCacheEntry{
		companyConfig: &config.CompanyConfig{CompanyID: 8},
		expiresAtUnix: time.Now().Add(companyConfigCacheTTL).UnixNano(),
	}

	sweepCompanyConfigCache()

	if _, stillThere := companyConfigCache[7]; stillThere {
		t.Fatal("the expired entry was kept")
	}
	if _, stillThere := companyConfigCache[8]; !stillThere {
		t.Fatal("a live entry was dropped")
	}
}

// A second write for the same company replaces the entry rather than adding one,
// which is what makes the write-through after a mutation an invalidation.
func TestCompanyConfigCacheWriteReplacesTheEntry(t *testing.T) {
	withEmptyConfigCache(t)
	writeCompanyConfigCache(42, &config.CompanyConfig{CompanyID: 42, SourceUpdated: 1})
	writeCompanyConfigCache(42, &config.CompanyConfig{CompanyID: 42, SourceUpdated: 2})

	if len(companyConfigCache) != 1 {
		t.Fatalf("expected one entry, got %v", len(companyConfigCache))
	}
	cached := readCompanyConfigCache(42)
	if cached == nil || cached.SourceUpdated != 2 {
		t.Fatal("the cache kept the older configuration")
	}
}
