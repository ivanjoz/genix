package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Ambos valores venían hardcodeados en deploy.sh: no salen de config.toml.
const (
	backupAWSProfile = "ivanjoz"
	backupS3Bucket   = "gerp-v2-frontend"
)

const (
	groupPublish        = "Publicar Código"
	groupProcess        = "Ejecutar Proceso"
	groupInfrastructure = "Infraestructura"
	groupLocal          = "Local Development"
)

var groupOrder = []string{groupPublish, groupProcess, groupInfrastructure, groupLocal}

// deployContext reúne todo lo que una acción o un script necesitan para ejecutarse.
type deployContext struct {
	repositoryRoot  string
	configFile      string
	goBinary        string
	companyID       string
	scriptArguments []string
}

type deployAction struct {
	id    int
	group string
	label string
	run   func(context deployContext) error
}

// deployActions está en el ORDEN DEL MENÚ, que no es el de ejecución (ver executionOrder).
var deployActions = []deployAction{
	{id: 1, group: groupPublish, label: "Frontend (Main + Store → docs/)", run: func(context deployContext) error {
		// 'publish' ya integra build-all.js (Main + Store) y postbuild.js (zip only bundled).
		if err := runCommand(context, "frontend", "bun", "run", "publish"); err != nil {
			return err
		}
		fmt.Println("✅ El bundle frontend.zip ha sido generado en ./docs!")
		fmt.Println("💡 Recuerde hacer git add docs/frontend.zip y push para activar el deploy.")
		return nil
	}},

	{id: 2, group: groupPublish, label: "Backend (AWS Cloud)", run: func(context deployContext) error {
		if err := generateRouteIDs(context); err != nil {
			return err
		}
		return runCommand(context, "cloud", context.goBinary, "run", ".", "accion=1")
	}},

	{id: 3, group: groupPublish, label: "Backend (VPS)", run: func(context deployContext) error {
		if err := generateRouteIDs(context); err != nil {
			return err
		}
		return runCommand(context, "scripts", context.goBinary, "run", ".", "deploy_vps")
	}},

	{id: 4, group: groupPublish, label: "Backup Lib (S3 Binary)", run: func(context deployContext) error {
		buildEnv := []string{"GOOS=linux", "GOARCH=arm64"}
		if err := runCommandWithEnv(context, "db-backup", buildEnv, context.goBinary, "build", "-ldflags", "-s -w", "."); err != nil {
			return err
		}
		return runCommand(context, "db-backup", "aws", "--profile", backupAWSProfile,
			"s3", "cp", "./db-backup", "s3://"+backupS3Bucket+"/_bin/db-backup.bin")
	}},

	{id: 5, group: groupProcess, label: "Desplegar Tablas (Backend)", run: deployTables},

	{id: 6, group: groupProcess, label: "Desplegar: Tablas, Datos Iniciales, Cloudflare Worker", run: func(context deployContext) error {
		if err := deployTables(context); err != nil {
			return err
		}
		fmt.Println("--- Poblando datos iniciales ---")
		if err := runCommand(context, "backend", context.goBinary, "run", ".", "fn-init"); err != nil {
			return err
		}
		fmt.Println("--- Desplegando Cloudflare Worker ---")
		return runCommand(context, "backend", context.goBinary, "run", ".", "fn-deploy-cloudflare-worker")
	}},

	{id: 7, group: groupProcess, label: "Inspeccionar/Compilar Backend", run: func(context deployContext) error {
		return runCommand(context, "backend", context.goBinary, "build", "-v", ".")
	}},

	{id: 10, group: groupProcess, label: "Deploy Cloudflare Worker", run: func(context deployContext) error {
		return runCommand(context, "backend", context.goBinary, "run", ".", "fn-deploy-cloudflare-worker")
	}},

	{id: 11, group: groupProcess, label: "Deploy Company Webpage", run: func(context deployContext) error {
		companyID, err := resolveCompanyID(context.companyID)
		if err != nil {
			return err
		}
		return runCommand(context, "backend", context.goBinary, "run", ".", "fn-deploy-company-webpage", companyID)
	}},

	{id: 12, group: groupProcess, label: "Sincronizar Catálogo de Imágenes", run: func(context deployContext) error {
		return runCommand(context, "backend", context.goBinary, "run", ".", "fn-sync-image-assets")
	}},

	{id: 13, group: groupProcess, label: "Actualizar Variables de Entorno de las Lambdas (AWS CLI)", run: updateLambdaEnvironmentVariables},

	{id: 14, group: groupProcess, label: "Configurar Variables Frontend en GitHub", run: func(context deployContext) error {
		return runCommand(context, ".", "bun", "run", "./scripts/set-github-frontend-vars.ts")
	}},

	{id: 9, group: groupInfrastructure, label: "Desplegar Infraestructura", run: func(context deployContext) error {
		return runCommand(context, "cloud", context.goBinary, "run", ".", "accion=3")
	}},

	{id: 8, group: groupLocal, label: "Serve Local Build (docs/)", run: func(context deployContext) error {
		if _, err := os.Stat(filepath.Join(context.repositoryRoot, "frontend", "build")); err != nil {
			return fmt.Errorf("la carpeta './frontend/build' no existe: ejecute el paso [1] primero")
		}
		fmt.Println("Iniciando servidor local en http://localhost:3000...")
		// serve.json dentro de build/ es el que resuelve el ruteo multi-SPA.
		return runCommand(context, ".", "bun", "x", "serve", "./frontend/build", "-l", "3000")
	}},
}

// executionOrder no es el orden del menú: la infraestructura (9) va antes que las tablas
// porque CloudFormation es dueño de la tabla DynamoDB que fn-init necesita. Es exactamente
// el orden en el que deploy.sh evaluaba sus bloques.
var executionOrder = []int{14, 1, 8, 2, 3, 4, 9, 5, 6, 10, 7, 11, 13, 12}

// generateRouteIDs corre antes de compilar el backend para que ninguna ruta llegue a producción
// sin número: sin él, una ruta nueva escribiría user_logs con route_id 0 hasta que alguien se
// acordara de regenerar. Escribe en vez de sólo verificar —fallar el deploy por un archivo
// generado no aporta nada cuando regenerarlo es determinista— pero avisa si cambió algo, porque
// el archivo hay que commitearlo.
func generateRouteIDs(context deployContext) error {
	fmt.Println("--- Regenerando IDs de rutas ---")
	if err := runCommand(context, "scripts", context.goBinary, "run", ".", "generate_route_ids"); err != nil {
		return err
	}
	fmt.Println("💡 Si api_routes.generated.go cambió, recuerde commitearlo.")
	return nil
}

// deployTables regenera controllers.generated.go antes de homologar para que fn-homologate
// vea todos los structs de tabla actuales.
func deployTables(context deployContext) error {
	fmt.Println("--- Regenerando controllers.generated.go ---")
	if err := runCommand(context, "scripts", context.goBinary, "run", ".", "generate_controllers"); err != nil {
		return err
	}
	fmt.Println("--- Recreando tablas ---")
	return runCommand(context, "backend", context.goBinary, "run", ".", "fn-homologate")
}

// actionScreen es lo que la pantalla de acciones necesita: las secciones a dibujar y el ID de
// acción de cada botón, en el mismo orden en que flattenButtons los recorre.
type actionScreen struct {
	sections  []buttonSection
	actionIDs []int
}

// newActionScreen arma los botones para un BACKEND_PROVIDER dado.
func newActionScreen(backendProvider string) actionScreen {
	var screen actionScreen

	for _, group := range groupOrder {
		section := buttonSection{title: group + " " + strings.Repeat("─", 6)}

		for _, action := range deployActions {
			if action.group != group {
				continue
			}
			label, disabled, hidden := backendButtonState(action.id, action.label, backendProvider)
			if hidden {
				continue
			}
			section.buttons = append(section.buttons, &Button{Label: label, Disabled: disabled})
			screen.actionIDs = append(screen.actionIDs, action.id)
		}

		if len(section.buttons) > 0 {
			screen.sections = append(screen.sections, section)
		}
	}

	return screen
}

// backendButtonState colapsa las acciones 2 (AWS Lambda) y 3 (VPS) en un único botón según el
// BACKEND_PROVIDER del entorno elegido: "aws" → Lambda, "none" o vacío → VPS. "cloudflare" no
// está soportado y se muestra deshabilitado en vez de omitirse, para que quede visible el
// motivo por el que no se puede publicar el backend.
func backendButtonState(actionID int, label, backendProvider string) (buttonLabel string, disabled, hidden bool) {
	isVPS := backendProvider == "none" || backendProvider == ""

	switch actionID {
	case 2:
		if isVPS {
			return "", false, true
		}
		if backendProvider == "cloudflare" {
			return "Backend (cloudflare no soportado)", true, false
		}
	case 3:
		if !isVPS {
			return "", false, true
		}
	}

	return label, false, false
}

func findAction(actionID int) *deployAction {
	for index := range deployActions {
		if deployActions[index].id == actionID {
			return &deployActions[index]
		}
	}
	return nil
}

// runSelectedActions ejecuta en executionOrder, no en el orden en que se marcaron los botones.
func runSelectedActions(context deployContext, selectedIDs []int) error {
	selected := map[int]bool{}
	for _, id := range selectedIDs {
		selected[id] = true
	}

	// git pull sólo cuando se va a publicar código, igual que deploy.sh. Un fallo no aborta:
	// puede haber cambios locales sin commitear y aun así querer publicar.
	if selected[1] || selected[2] || selected[3] || selected[4] {
		fmt.Println("Obteniendo los últimos cambios del repositorio (GIT PULL)...")
		if err := runCommand(context, ".", "git", "pull"); err != nil {
			fmt.Printf("⚠️  git pull falló: %v\n", err)
		}
	}

	for _, actionID := range executionOrder {
		if !selected[actionID] {
			continue
		}
		// La acción 6 ya recrea las tablas, así que correr también la 5 sería trabajo repetido.
		if actionID == 5 && selected[6] {
			continue
		}

		action := findAction(actionID)
		fmt.Printf("\n=== [%d] %s ===\n", action.id, action.label)
		if err := action.run(context); err != nil {
			return fmt.Errorf("acción %d (%s): %w", action.id, action.label, err)
		}
	}

	return nil
}

// runCommand ejecuta heredando stdio para que la salida se vea igual que con deploy.sh.
// workingDirectory es relativo a la raíz del repositorio.
func runCommand(context deployContext, workingDirectory, name string, args ...string) error {
	return runCommandWithEnv(context, workingDirectory, nil, name, args...)
}

func runCommandWithEnv(context deployContext, workingDirectory string, extraEnv []string, name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Dir = filepath.Join(context.repositoryRoot, workingDirectory)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	command.Env = append(os.Environ(), extraEnv...)
	return command.Run()
}

var positiveIntegerPattern = regexp.MustCompile(`^[1-9][0-9]*$`)

// resolveCompanyID usa el ID recibido por argumento o, si no vino, lo pide por stdin.
func resolveCompanyID(companyIDArgument string) (string, error) {
	companyID := strings.TrimSpace(companyIDArgument)
	if companyID == "" {
		fmt.Print("Ingrese CompanyID: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("no se pudo leer el CompanyID: %w", err)
		}
		companyID = strings.TrimSpace(line)
	}

	if !positiveIntegerPattern.MatchString(companyID) {
		return "", fmt.Errorf("CompanyID debe ser un entero positivo, se recibió: %q", companyID)
	}
	return companyID, nil
}
