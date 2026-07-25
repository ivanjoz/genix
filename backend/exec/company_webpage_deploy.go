package exec

import (
	"app/cloud"
	configTypes "app/config/types"
	"app/core"
	"app/db"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

const (
	webpageConfigGroup            = int32(10)
	webpageAssetUploadConcurrency = 4
)

func DeployCompanyWebpage(args *core.ExecArgs) core.FuncResponse {
	companyID, argumentError := parseCompanyIDArgument(args.Message)
	if argumentError != nil {
		return args.MakeErr(argumentError)
	}

	hostname, domainError := getCompanyWebpageDomain(companyID)
	if domainError != nil {
		return args.MakeErr(domainError)
	}

	projectRoot, rootError := findGenixProjectRoot()
	if rootError != nil {
		return args.MakeErr(rootError)
	}

	webpageDirectory := filepath.Join(projectRoot, "frontend", "webpage")
	webpagesDirectory := filepath.Join(webpageDirectory, "cloudflare", "webpages")
	targetDirectory := filepath.Join(webpagesDirectory, hostname)
	temporaryDirectory := filepath.Join(webpagesDirectory, ".build-"+hostname)
	assetBase, assetBaseError := companyWebpageAssetBase(companyID)
	if assetBaseError != nil {
		return args.MakeErr(assetBaseError)
	}
	if corsError := ensureCompanyWebpageAssetCORS(projectRoot); corsError != nil {
		return args.MakeErr(corsError)
	}

	fmt.Printf("[company-webpage] company=%d hostname=%s\n", companyID, hostname)
	fmt.Printf("[company-webpage] build=%s\n", temporaryDirectory)
	fmt.Printf("[company-webpage] assets=%s\n", assetBase)

	if removeError := os.RemoveAll(temporaryDirectory); removeError != nil {
		return args.MakeErr("error limpiando build temporal:", removeError)
	}

	// The script derives the asset base from credentials.json (FRONTEND_CDN +
	// /websites/<company>, same as companyWebpageAssetBase), so no --asset-base is passed.
	prerenderCommand := exec.Command(
		"bun",
		"scripts/prerender.mjs",
		"--company",
		strconv.FormatInt(int64(companyID), 10),
		"--out",
		temporaryDirectory,
	)
	prerenderCommand.Dir = projectRoot
	prerenderCommand.Stdout = os.Stdout
	prerenderCommand.Stderr = os.Stderr
	if prerenderError := prerenderCommand.Run(); prerenderError != nil {
		return args.MakeErr("error generando storefront:", prerenderError)
	}

	if verificationError := verifyGeneratedWebpage(temporaryDirectory); verificationError != nil {
		return args.MakeErr(verificationError)
	}
	if uploadError := uploadCompanyWebpageAssets(companyID, temporaryDirectory); uploadError != nil {
		return args.MakeErr(uploadError)
	}
	if splitError := removeUploadedWebpageAssets(temporaryDirectory); splitError != nil {
		return args.MakeErr(splitError)
	}
	if swapError := replaceWebpageDirectory(targetDirectory, temporaryDirectory); swapError != nil {
		return args.MakeErr("error publicando build local:", swapError)
	}

	if _, deployError := DeployCloudflareWorker(); deployError != nil {
		return args.MakeErr(deployError)
	}
	if provisionError := provisionStorefrontDomain(hostname); provisionError != nil {
		return args.MakeErr(provisionError)
	}

	return core.FuncResponse{
		Message: fmt.Sprintf("Webpage de CompanyID %d desplegada en https://%s", companyID, hostname),
	}
}

func companyWebpageAssetBase(companyID int32) (string, error) {
	frontendCDN := strings.TrimRight(strings.TrimSpace(core.Env.FRONTEND_CDN), "/")
	if !strings.HasPrefix(frontendCDN, "https://") {
		return "", fmt.Errorf("FRONTEND_CDN debe ser una URL https válida")
	}
	return fmt.Sprintf("%s/websites/%d", frontendCDN, companyID), nil
}

func uploadCompanyWebpageAssets(companyID int32, webpageDirectory string) error {
	// Upload assets before HTML so a published page never references missing modules.
	entries, readError := os.ReadDir(webpageDirectory)
	if readError != nil {
		return fmt.Errorf("error leyendo assets del storefront: %w", readError)
	}

	assetPath := fmt.Sprintf("websites/%d", companyID)
	assetEntries := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.EqualFold(filepath.Ext(entry.Name()), ".html") || entry.Name() == "sw.js" {
			continue
		}
		assetEntries = append(assetEntries, entry)
	}
	if len(assetEntries) == 0 {
		return fmt.Errorf("el prerender no generó assets para R2")
	}

	uploadGroup := errgroup.Group{}
	uploadGroup.SetLimit(webpageAssetUploadConcurrency)
	for _, entry := range assetEntries {
		entry := entry
		uploadGroup.Go(func() error {
			contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(entry.Name())))
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			cacheControl := "public, max-age=300"
			if isFingerprintedWebpageAsset(entry.Name()) {
				cacheControl = "public, max-age=31536000, immutable"
			}

			fmt.Printf("[company-webpage] uploading asset=%s/%s\n", assetPath, entry.Name())
			if uploadError := cloud.SaveFile(cloud.SaveFileArgs{
				Path:          assetPath,
				Name:          entry.Name(),
				LocalFilePath: filepath.Join(webpageDirectory, entry.Name()),
				ContentType:   contentType,
				CacheControl:  cacheControl,
			}); uploadError != nil {
				return fmt.Errorf("error subiendo asset %s: %w", entry.Name(), uploadError)
			}
			return nil
		})
	}
	if uploadError := uploadGroup.Wait(); uploadError != nil {
		return uploadError
	}

	fmt.Printf("[company-webpage] uploaded-assets=%d path=%s\n", len(assetEntries), assetPath)
	return nil
}

func isFingerprintedWebpageAsset(fileName string) bool {
	if fileName == "env.js" || fileName == "blurhash.js" {
		return false
	}
	nameWithoutExtension := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	nameParts := strings.Split(nameWithoutExtension, ".")
	return len(nameParts[len(nameParts)-1]) >= 8
}

func removeUploadedWebpageAssets(webpageDirectory string) error {
	// Keep only HTML and the same-origin service worker in Workers Static Assets.
	entries, readError := os.ReadDir(webpageDirectory)
	if readError != nil {
		return readError
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.EqualFold(filepath.Ext(entry.Name()), ".html") || entry.Name() == "sw.js" {
			continue
		}
		if removeError := os.Remove(filepath.Join(webpageDirectory, entry.Name())); removeError != nil {
			return fmt.Errorf("error removiendo asset local %s: %w", entry.Name(), removeError)
		}
	}
	return nil
}

func parseCompanyIDArgument(rawArgument string) (int32, error) {
	arguments := strings.Fields(rawArgument)
	if len(arguments) != 1 {
		return 0, fmt.Errorf("fn-deploy-company-webpage requiere exactamente un CompanyID")
	}

	companyID64, parseError := strconv.ParseInt(arguments[0], 10, 32)
	if parseError != nil || companyID64 <= 0 {
		return 0, fmt.Errorf("CompanyID debe ser un entero positivo")
	}

	return int32(companyID64), nil
}

func getCompanyWebpageDomain(companyID int32) (string, error) {
	parameters := []configTypes.Parameters{}
	query := db.Query(&parameters).CompanyID.Equals(companyID)
	query.Group.Equals(webpageConfigGroup)
	query.Key.Equals("domain")
	if queryError := query.Exec(); queryError != nil {
		return "", fmt.Errorf("error consultando el dominio de CompanyID %d: %w", companyID, queryError)
	}

	activeDomains := []string{}
	for _, parameter := range parameters {
		if parameter.Status > 0 && strings.TrimSpace(parameter.Value) != "" {
			activeDomains = append(activeDomains, parameter.Value)
		}
	}
	if len(activeDomains) != 1 {
		return "", fmt.Errorf(
			"CompanyID %d debe tener exactamente un dominio activo; encontrados: %d",
			companyID,
			len(activeDomains),
		)
	}

	hostname, hostnameError := normalizeAndValidateHostname(activeDomains[0])
	if hostnameError != nil {
		return "", hostnameError
	}
	return hostname, nil
}

func verifyGeneratedWebpage(webpageDirectory string) error {
	if !fileExists(filepath.Join(webpageDirectory, "index.html")) {
		return fmt.Errorf("el prerender no generó index.html")
	}

	entries, readError := os.ReadDir(webpageDirectory)
	if readError != nil {
		return readError
	}
	for _, entry := range entries {
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if !entry.IsDir() && (extension == ".js" || extension == ".css") {
			return nil
		}
	}

	return fmt.Errorf("el prerender no generó assets JS/CSS")
}

func replaceWebpageDirectory(targetDirectory string, temporaryDirectory string) error {
	backupDirectory := targetDirectory + ".previous"
	if removeBackupError := os.RemoveAll(backupDirectory); removeBackupError != nil {
		return removeBackupError
	}

	targetExists := false
	if _, targetError := os.Stat(targetDirectory); targetError == nil {
		targetExists = true
		if backupError := os.Rename(targetDirectory, backupDirectory); backupError != nil {
			return backupError
		}
	} else if !os.IsNotExist(targetError) {
		return targetError
	}

	if publishError := os.Rename(temporaryDirectory, targetDirectory); publishError != nil {
		if targetExists {
			_ = os.Rename(backupDirectory, targetDirectory)
		}
		return publishError
	}

	return os.RemoveAll(backupDirectory)
}
