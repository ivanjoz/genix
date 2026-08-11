package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Los grupos de la pestaña Scripts.
const (
	scriptGroupDatabase   = "Base de Datos"
	scriptGroupServers    = "Servidores"
	scriptGroupGenerators = "Generadores"
)

var scriptGroupOrder = []string{scriptGroupDatabase, scriptGroupServers, scriptGroupGenerators}

// scriptEntry es un utilitario de los que hoy despacha app.sh. Cada entrada replica el mismo
// comando que ese script ejecuta, porque la idea es que app.sh desaparezca.
type scriptEntry struct {
	key   string
	group string
	label string
	// argumentsHint, cuando no está vacío, indica que el script necesita argumentos: se piden
	// por stdin antes de ejecutarlo si no vinieron por línea de comandos.
	argumentsHint string
	run           func(context deployContext, arguments []string) error
}

var deployScripts = []scriptEntry{
	{key: "check_tables", group: scriptGroupDatabase, label: "Validar Tablas", run: func(context deployContext, _ []string) error {
		return runCommand(context, "scripts", context.goBinary, "run", ".", "check_tables")
	}},

	{key: "create", group: scriptGroupDatabase, label: "Crear Tabla",
		argumentsHint: "<output_path> <table_name> [campo:tipo:key]...",
		run: func(context deployContext, arguments []string) error {
			return runTableScript(context, "create", arguments)
		}},

	{key: "edit", group: scriptGroupDatabase, label: "Editar Tabla (agregar columna)",
		argumentsHint: "<table_name> <campo:tipo[:key]>",
		run: func(context deployContext, arguments []string) error {
			return runTableScript(context, "edit", arguments)
		}},

	// Los tres configuradores preguntan su propio modo de instalación, así que no se les
	// piden argumentos acá.
	{key: "configure_server", group: scriptGroupServers, label: "Configurar Servidor (systemd + Nginx)", run: func(context deployContext, _ []string) error {
		return runCommand(context, ".", "python3", "scripts/configure_server.py")
	}},

	{key: "configure_db", group: scriptGroupServers, label: "Configurar Base de Datos (ScyllaDB + GenixSearch + Qdrant)", run: func(context deployContext, _ []string) error {
		return runCommand(context, ".", "python3", "scripts/configure_db.py")
	}},

	{key: "configure_server_utils", group: scriptGroupServers, label: "Configurar Server Utils (rate limiter + SSE bridge)", run: func(context deployContext, _ []string) error {
		return runCommand(context, ".", "python3", "scripts/configure_server_utils.py")
	}},

	{key: "follow_cloudwatch_logs", group: scriptGroupServers, label: "Follow Cloudwatch Logs", run: func(context deployContext, _ []string) error {
		return runCommand(context, "scripts", context.goBinary, "run", ".", "follow_cloudwatch_logs")
	}},

	{key: "generate_controllers", group: scriptGroupGenerators, label: "Regenerar Controllers", run: func(context deployContext, _ []string) error {
		return runCommand(context, "scripts", context.goBinary, "run", ".", "generate_controllers")
	}},

	{key: "sync_struct_interfaces", group: scriptGroupGenerators, label: "Sincronizar Interfaces del Frontend", run: func(context deployContext, _ []string) error {
		return runCommand(context, "scripts", context.goBinary, "run", ".", "sync_struct_interfaces")
	}},

	{key: "generate_menu_descriptions", group: scriptGroupGenerators, label: "Exportar Descripciones del Menú", run: func(context deployContext, _ []string) error {
		return runCommand(context, "scripts", context.goBinary, "run", ".", "generate_menu_descriptions")
	}},

	{key: "index_documentation", group: scriptGroupGenerators, label: "Validar / Indexar Documentación RAG",
		argumentsHint: "-mode validate|index [-dry-run] [-qdrant-host HOST] [-document PATH]",
		run: func(context deployContext, arguments []string) error {
			return runCommand(context, "backend", context.goBinary, append([]string{"run", "./agent/cmd/documentation-index"}, arguments...)...)
		}},

	{key: "search_documentation", group: scriptGroupGenerators, label: "Buscar Documentación RAG",
		argumentsHint: "-question TEXTO|-examples [-limit N] [-qdrant-host HOST]",
		run: func(context deployContext, arguments []string) error {
			return runCommand(context, "backend", context.goBinary, append([]string{"run", "./agent/cmd/documentation-search"}, arguments...)...)
		}},

	{key: "classify_agent_request", group: scriptGroupGenerators, label: "Evaluar Clasificador del Agente",
		argumentsHint: "-question TEXTO [-surface KIND] [-route PATH] [-language es|en] [-selected-section ID]",
		run: func(context deployContext, arguments []string) error {
			return runCommand(context, "backend", context.goBinary, append([]string{"run", "./agent/cmd/classifier-eval"}, arguments...)...)
		}},

	{key: "generate_sale_orders", group: scriptGroupGenerators, label: "Generar Órdenes de Venta (demo)", run: func(context deployContext, _ []string) error {
		return runCommand(context, "backend", context.goBinary, "run", ".", "fn-generate-sale-orders")
	}},
}

func runTableScript(context deployContext, command string, arguments []string) error {
	goArguments := append([]string{"run", "table/create_edit_table.go", command}, arguments...)
	return runCommand(context, "scripts", context.goBinary, goArguments...)
}

// scriptScreen es la pestaña Scripts: las secciones a dibujar y la clave de cada botón, en el
// mismo orden en que flattenButtons los recorre.
type scriptScreen struct {
	sections   []buttonSection
	scriptKeys []string
}

func newScriptScreen() scriptScreen {
	var screen scriptScreen

	for _, group := range scriptGroupOrder {
		section := buttonSection{title: group + " " + strings.Repeat("─", 6)}
		for _, script := range deployScripts {
			if script.group != group {
				continue
			}
			section.buttons = append(section.buttons, &Button{Label: script.label})
			screen.scriptKeys = append(screen.scriptKeys, script.key)
		}
		if len(section.buttons) > 0 {
			screen.sections = append(screen.sections, section)
		}
	}

	return screen
}

func findScript(scriptKey string) *scriptEntry {
	for index := range deployScripts {
		if deployScripts[index].key == scriptKey {
			return &deployScripts[index]
		}
	}
	return nil
}

// runSelectedScripts ejecuta los utilitarios marcados, en el orden del menú.
func runSelectedScripts(context deployContext, selectedKeys []string) error {
	for _, scriptKey := range selectedKeys {
		script := findScript(scriptKey)
		fmt.Printf("\n=== %s ===\n", script.label)

		arguments := context.scriptArguments
		if script.argumentsHint != "" && len(arguments) == 0 {
			var err error
			if arguments, err = askScriptArguments(*script); err != nil {
				return err
			}
		}

		if err := script.run(context, arguments); err != nil {
			return fmt.Errorf("script %s (%s): %w", script.key, script.label, err)
		}
	}
	return nil
}

func askScriptArguments(script scriptEntry) ([]string, error) {
	fmt.Printf("Argumentos para '%s' %s: ", script.key, script.argumentsHint)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("no se pudieron leer los argumentos de %s: %w", script.key, err)
	}
	return strings.Fields(strings.TrimSpace(line)), nil
}
