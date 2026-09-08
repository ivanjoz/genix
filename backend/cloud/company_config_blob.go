// A company's whole configuration, sealed into one object in storage and treated
// as a live cache: written through on every mutation, rebuilt by whoever finds it
// missing. An API that needs a company's RUC, SOL credentials, signing key, series
// or parameters does one GET instead of four reads across three tables.
//
// This lives in `cloud` and not in config/types because it needs both: the record
// (L2) and object storage (L3). L2 may import nothing above itself, so the only
// legal home for the pair is here (docs/MODULE_BOUNDARIES.md).

package cloud

import (
	config "app/config/types"
	"app/core"
	"app/db"
	invoicing "app/invoicing/types"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ivanjoz/colbin"
	"github.com/ivanjoz/fareward/go/siphash"
)

const (
	// companyConfigDomain separates this use of the project secret from every other
	// one, and doubles as the format version: bumping it retires every existing blob
	// at once, and readers rebuild on the miss.
	companyConfigDomain = "company-config-v1"
	companyConfigPath   = "company-config"
	// aesGCMNonceSize is 12 bytes, the size GCM is specified for.
	aesGCMNonceSize = 12
)

// companyConfigObjectName is the object key for a company: c42_cf_1kQ9x3FfA2w.cb
//
// The tail is the fareward MAC strategy — SipHash-2-4 keyed on the project secret,
// domain-separated, 64 bits big-endian — so it is recomputable by any service
// holding the secret and unguessable without it. Bucket listing or id enumeration
// reveals nothing, because the name of a company's blob is itself a secret.
func companyConfigObjectName(companyID int32) string {
	hasher := siphash.New(siphash.DeriveKey([]byte(core.Env.SECRET_PHRASE)))
	hasher.WriteString(companyConfigDomain)
	companyBytes := [4]byte{}
	binary.BigEndian.PutUint32(companyBytes[:], uint32(companyID))
	hasher.Write(companyBytes[:])

	tag := binary.BigEndian.AppendUint64(make([]byte, 0, 8), hasher.Sum64())
	return fmt.Sprintf("c%v_cf_%v.cb", companyID, base64.RawURLEncoding.EncodeToString(tag))
}

// companyConfigCipherKey is 16 bytes: AES-128, not the AES-256 core.Encrypt uses.
// Derived rather than taken raw because SECRET_PHRASE is a configuration string of
// any length, so every byte of it has to reach the key.
func companyConfigCipherKey() ([]byte, error) {
	if len(core.Env.SECRET_PHRASE) == 0 {
		return nil, errors.New("no hay SECRET_PHRASE configurada para cifrar la configuración de la empresa")
	}
	digest := sha256.Sum256([]byte(core.Env.SECRET_PHRASE + companyConfigDomain))
	return digest[:16], nil
}

// sealCompanyConfig encrypts with AES-128-GCM, nonce prepended.
//
// The object name is the additional authenticated data, which binds a blob to its
// filename: a blob copied over another company's object fails to open instead of
// decrypting into the wrong company's credentials.
func sealCompanyConfig(payload []byte, objectName string) ([]byte, error) {
	key, err := companyConfigCipherKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCMNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("no se pudo generar el nonce: %w", err)
	}
	return append(nonce, aesgcm.Seal(nil, nonce, payload, []byte(objectName))...), nil
}

func openCompanyConfig(sealed []byte, objectName string) ([]byte, error) {
	if len(sealed) <= aesGCMNonceSize {
		return nil, errors.New("el archivo de configuración está truncado")
	}
	key, err := companyConfigCipherKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return aesgcm.Open(nil, sealed[:aesGCMNonceSize], sealed[aesGCMNonceSize:], []byte(objectName))
}

// CompactAndStoreCompanyConfig reads a company whole, seals it and stores it.
//
// Call it after a mutation has committed — not before, and never in a way that can
// fail the caller's request: the save the user asked for already happened, so
// refusing it because an upload failed would be a lie about what was written. The
// next save, or the first reader, repairs the blob.
func CompactAndStoreCompanyConfig(companyID int32) (*config.CompanyConfig, error) {
	if companyID == 0 {
		return nil, errors.New("no se envió el ID de la empresa")
	}

	company, err := readCompanyForConfig(companyID)
	if err != nil {
		return nil, err
	}
	secrets, err := readActiveSecretsForConfig(companyID)
	if err != nil {
		return nil, err
	}
	parameters, err := readParametersForConfig(companyID)
	if err != nil {
		return nil, err
	}

	companyConfig, err := config.AssembleCompanyConfig(company, secrets, parameters)
	if err != nil {
		return nil, err
	}

	payload, err := colbin.Marshal(companyConfig)
	if err != nil {
		return nil, fmt.Errorf("no se pudo serializar la configuración de la empresa: %w", err)
	}

	objectName := companyConfigObjectName(companyID)
	sealed, err := sealCompanyConfig(payload, objectName)
	if err != nil {
		return nil, err
	}

	err = SaveFile(SaveFileArgs{
		Name:        objectName,
		Path:        companyConfigPath,
		FileContent: sealed,
		ContentType: "application/octet-stream",
		// The blob changes whenever the company does, and every reader wants the
		// current one — a cached copy at a CDN edge would serve retired credentials.
		CacheControl: "no-store",
	})
	if err != nil {
		return nil, fmt.Errorf("no se pudo guardar la configuración de la empresa: %w", err)
	}

	// Cached here rather than only on the read path, so the write-through that follows
	// a mutation replaces this process's entry instead of leaving it to expire: the
	// instance that saved the certificate is the one most likely to be asked next.
	writeCompanyConfigCache(companyID, companyConfig)
	return companyConfig, nil
}

// StoreCompanyConfigAsync is the write-through call sites use: it never returns an
// error, because none of them may fail on it.
func StoreCompanyConfigAsync(companyID int32) {
	if _, err := CompactAndStoreCompanyConfig(companyID); err != nil {
		core.Log("Error al compactar la configuración de la empresa", companyID, ":", err)
	}
}

// The process-local half of the cache, in front of the object in storage.
//
// A single emission asks for the config once, but a till issuing a run of documents
// or a retry sweep working through a backlog asks for the same company over and
// over, and each of those was a GET, a decrypt and an unmarshal. Twenty seconds is
// short enough that a rotated certificate or a retired series takes effect while
// somebody is still looking at the screen that changed it.
const companyConfigCacheTTL = 20 * time.Second

// companyConfigCacheEntry keeps the deadline as unix nanoseconds rather than a
// time.Time: 8 bytes instead of 24, and one comparison instead of a method call.
//
// The config is a pointer because that is what both sides of this cache already
// hold — CompactAndStoreCompanyConfig built one, LoadCompanyConfig returns one — and
// a value would copy ~400 bytes in and out on every read while still sharing the key
// and certificate slices, so the isolation would be partial: safe-looking without
// being safe. What comes out is read-only instead, which is stated where it is read.
type companyConfigCacheEntry struct {
	companyConfig *config.CompanyConfig
	expiresAtUnix int64
}

var (
	companyConfigCache       = map[int32]companyConfigCacheEntry{}
	companyConfigCacheMutex  sync.RWMutex
	companyConfigSweeperOnce sync.Once
)

// readCompanyConfigCache returns the live entry for a company, or nil.
//
// What comes out is shared with every other caller holding it, so it is read-only.
// BuildIssuer copies the fields it needs into an Issuer and nothing writes back;
// deep-copying here would mean copying the private key and the certificate on every
// read, which is the work this cache exists to avoid.
func readCompanyConfigCache(companyID int32) *config.CompanyConfig {
	companyConfigCacheMutex.RLock()
	defer companyConfigCacheMutex.RUnlock()

	entry, isCached := companyConfigCache[companyID]
	if !isCached || time.Now().UnixNano() > entry.expiresAtUnix {
		return nil
	}
	return entry.companyConfig
}

// writeCompanyConfigCache stores a company's configuration, and starts the sweeper
// the first time anything is cached.
//
// Started from here and not from main because this package is linked into the
// Lambda, into every script and into the test binaries: a ticker that exists in a
// process which never cached a company is a goroutine no deployment asked for.
//
// time.Now, not core.Now: this is a real twenty-second timer, and the effective
// clock can be frozen — which would expire either nothing or everything.
func writeCompanyConfigCache(companyID int32, companyConfig *config.CompanyConfig) {
	companyConfigSweeperOnce.Do(startCompanyConfigCacheSweeper)

	companyConfigCacheMutex.Lock()
	defer companyConfigCacheMutex.Unlock()

	companyConfigCache[companyID] = companyConfigCacheEntry{
		companyConfig: companyConfig,
		expiresAtUnix: time.Now().Add(companyConfigCacheTTL).UnixNano(),
	}
}

// startCompanyConfigCacheSweeper drops expired entries on a fixed cadence, so an
// entry nobody asks for again does not hold a private key and a certificate for the
// life of the process.
//
// The ticker is never stopped: it lives as long as the process that has a cache to
// sweep, and stopping it would leave entries resident with nothing to remove them.
func startCompanyConfigCacheSweeper() {
	ticker := time.NewTicker(companyConfigCacheTTL)
	go func() {
		for range ticker.C {
			sweepCompanyConfigCache()
		}
	}()
}

// sweepCompanyConfigCache deletes every entry past its TTL.
//
// It runs on the tick, so an entry is held for up to twice the TTL before its memory
// is released — expiring at 20 seconds, swept by the tick that follows. It is never
// served in that window: readCompanyConfigCache checks the deadline itself, which is
// what keeps the sweep a memory concern and not a correctness one.
func sweepCompanyConfigCache() {
	companyConfigCacheMutex.Lock()
	defer companyConfigCacheMutex.Unlock()

	now := time.Now().UnixNano()
	for cachedID, entry := range companyConfigCache {
		if now > entry.expiresAtUnix {
			delete(companyConfigCache, cachedID)
		}
	}
}

// LoadCompanyConfig returns a company's configuration, from memory when it was read
// in the last companyConfigCacheTTL, from storage otherwise, and rebuilding it when
// the object is missing, unreadable or of a format this build does not know.
//
// That fallback is what makes the cache safe to be wrong: a missed write-through, a
// rotated secret or a version bump costs one rebuild, never a failed emission.
func LoadCompanyConfig(companyID int32) (*config.CompanyConfig, error) {
	if companyID == 0 {
		return nil, errors.New("no se envió el ID de la empresa")
	}
	if cached := readCompanyConfigCache(companyID); cached != nil {
		return cached, nil
	}
	objectName := companyConfigObjectName(companyID)

	sealed, err := GetFile(SaveFileArgs{Name: objectName, Path: companyConfigPath})
	if err == nil && len(sealed) > 0 {
		payload, openErr := openCompanyConfig(sealed, objectName)
		if openErr == nil {
			companyConfig := config.CompanyConfig{}
			if colbin.Unmarshal(payload, &companyConfig) == nil &&
				companyConfig.Version == config.CompanyConfigVersion &&
				companyConfig.CompanyID == companyID {
				writeCompanyConfigCache(companyID, &companyConfig)
				return &companyConfig, nil
			}
		}
	}

	// Not cached here: CompactAndStoreCompanyConfig caches what it built itself.
	return CompactAndStoreCompanyConfig(companyID)
}

// readCompanyForConfig mirrors invoicing's own company read: through the data
// mirror where one is configured, straight at the cluster otherwise.
func readCompanyForConfig(companyID int32) (*config.Company, error) {
	if IsDataMirrorEnabled() {
		company, err := GetByID(config.Company{ID: companyID})
		if err != nil {
			return nil, fmt.Errorf("error al leer la empresa: %w", err)
		}
		if company == nil {
			return nil, errors.New("no se encontró la empresa")
		}
		return company, nil
	}

	companies := []config.Company{}
	query := db.Query(&companies)
	query.ID.Equals(companyID).Limit(1)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer la empresa: %w", err)
	}
	if len(companies) == 0 {
		return nil, errors.New("no se encontró la empresa")
	}
	return &companies[0], nil
}

// readActiveSecretsForConfig returns the newest active SUNAT credentials, or nil.
//
// Nil is not an error: a company that has not uploaded a certificate still has a
// configuration worth caching, it just cannot issue yet.
func readActiveSecretsForConfig(companyID int32) (*invoicing.CompanySecrets, error) {
	secrets := []invoicing.CompanySecrets{}
	query := db.Query(&secrets)
	query.Select().CompanyID.Equals(companyID).Type.Equals(invoicing.SecretTypeSunatCPE)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las credenciales SUNAT: %w", err)
	}

	var active *invoicing.CompanySecrets
	for index := range secrets {
		candidate := &secrets[index]
		if candidate.Status != 1 {
			continue
		}
		// Highest id wins: rotation inserts a new row and retires the old one, so the
		// newest active row is the certificate in force.
		if active == nil || candidate.ID > active.ID {
			active = candidate
		}
	}
	return active, nil
}

func readParametersForConfig(companyID int32) ([]config.Parameters, error) {
	parameters := []config.Parameters{}
	query := db.Query(&parameters).CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los parámetros de la empresa: %w", err)
	}
	return parameters, nil
}
