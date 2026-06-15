package exec

import (
	"app/cloud"
	"app/core"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

// The base template is COMPANY-AGNOSTIC: --company only flips SSR/prerender on for the
// build (see scripts/prerender.mjs --page-base). Any positive id works — its value never
// scopes the output (always websites/base) and is only a placeholder default in the
// emitted cdn-url/page-id metas, which downstream tooling overwrites per tenant/page.
const baseTemplateBuildCompanyID = 1

// DeployBaseTemplate builds the company-agnostic storefront TEMPLATE (the /base shell: a
// spinner + an early-fetch <head> script driven entirely by its cdn-url/page-id metas)
// and uploads every emitted file to <FRONTEND_CDN>/websites/base. Downstream tooling
// fetches this template, rewrites those two metas, and serves the copy from any domain to
// render any tenant/page — so the HTML lives in the CDN alongside its js/css.
func DeployBaseTemplate(args *core.ExecArgs) core.FuncResponse {
	projectRoot, rootError := findGenixProjectRoot()
	if rootError != nil {
		return args.MakeErr(rootError)
	}

	webpageDirectory := filepath.Join(projectRoot, "frontend", "webpage")
	buildDirectory := filepath.Join(webpageDirectory, "cloudflare", "webpages", ".build-base")

	// Allow the template's js/css to be fetched cross-origin (it is served from other domains).
	if corsError := ensureCompanyWebpageAssetCORS(projectRoot); corsError != nil {
		return args.MakeErr(corsError)
	}
	if removeError := os.RemoveAll(buildDirectory); removeError != nil {
		return args.MakeErr("error limpiando build temporal:", removeError)
	}

	fmt.Printf("[base-template] build=%s\n", buildDirectory)

	// --page-base prerenders only routes/base/+page.svelte; --out overrides the default
	// dist-prerender/base so the upload reads from a known temp dir.
	prerenderCommand := exec.Command(
		"bun",
		"scripts/prerender.mjs",
		"--company", strconv.Itoa(baseTemplateBuildCompanyID),
		"--page-base",
		"--out", buildDirectory,
	)
	prerenderCommand.Dir = projectRoot
	prerenderCommand.Stdout = os.Stdout
	prerenderCommand.Stderr = os.Stderr
	if prerenderError := prerenderCommand.Run(); prerenderError != nil {
		return args.MakeErr("error generando template base:", prerenderError)
	}

	if verificationError := verifyGeneratedWebpage(buildDirectory); verificationError != nil {
		return args.MakeErr(verificationError)
	}
	if uploadError := uploadBaseTemplateAssets(buildDirectory); uploadError != nil {
		return args.MakeErr(uploadError)
	}
	if removeError := os.RemoveAll(buildDirectory); removeError != nil {
		return args.MakeErr("error limpiando build temporal:", removeError)
	}

	return core.FuncResponse{
		Message: "Template base desplegado en /websites/base del CDN",
	}
}

// uploadBaseTemplateAssets uploads EVERY file (the template index.html plus its js/css)
// to websites/base. Unlike the per-company deploy, the HTML lives in the CDN too because
// downstream tooling fetches it from there to copy + rewrite. Fingerprinted assets get an
// immutable year cache; the template HTML stays short-lived so edits propagate.
func uploadBaseTemplateAssets(buildDirectory string) error {
	entries, readError := os.ReadDir(buildDirectory)
	if readError != nil {
		return fmt.Errorf("error leyendo archivos del template base: %w", readError)
	}

	const assetPath = "websites/base"
	fileEntries := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			fileEntries = append(fileEntries, entry)
		}
	}
	if len(fileEntries) == 0 {
		return fmt.Errorf("el prerender no generó archivos para el template base")
	}

	uploadGroup := errgroup.Group{}
	uploadGroup.SetLimit(webpageAssetUploadConcurrency)
	for _, entry := range fileEntries {
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

			fmt.Printf("[base-template] uploading file=%s/%s\n", assetPath, entry.Name())
			if uploadError := cloud.SaveFile(cloud.SaveFileArgs{
				Path:          assetPath,
				Name:          entry.Name(),
				LocalFilePath: filepath.Join(buildDirectory, entry.Name()),
				ContentType:   contentType,
				CacheControl:  cacheControl,
			}); uploadError != nil {
				return fmt.Errorf("error subiendo archivo %s: %w", entry.Name(), uploadError)
			}
			return nil
		})
	}
	if uploadError := uploadGroup.Wait(); uploadError != nil {
		return uploadError
	}

	fmt.Printf("[base-template] uploaded-files=%d path=%s\n", len(fileEntries), assetPath)
	return nil
}
