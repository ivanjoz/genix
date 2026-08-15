# Deployer (TUI)

Reemplaza al antiguo `deploy.sh` en bash. La lógica vive en `scripts/deployer/` (Go + Bubble
Tea v2) y `deploy.sh` quedó como un wrapper de tres líneas.

## Uso

```bash
./deploy.sh              # interfaz de botones
./deploy.sh 6 9          # acciones 6 y 9, sin interfaz
./deploy.sh 11 42        # acción 11 para el CompanyID 42
./app.sh deploy          # equivalente a ./deploy.sh
```

Con argumentos no se muestra la interfaz. Los IDs pueden ir separados por espacios o comas.
Un token que no sea un ID de acción conocido se toma como `CompanyID` (acción 11), así que
`6 9` siguen siendo dos acciones. Si ningún token es una acción válida, el programa falla en
vez de abrir la interfaz.

## Navegación

Tres secciones, con `tab`/`shift+tab` o click en la barra superior:

| Sección | Contenido |
|---|---|
| **Environment** | El archivo de configuración. Elegir uno lo marca y lleva directo a Actions. |
| **Actions** | Los despliegues (lo que hacía `deploy.sh`). |
| **Scripts** | Los utilitarios que hoy despacha `app.sh`. |

Se abre en **Environment** cuando existe `config.1.toml` y en **Actions** cuando no, que es
cuando no hay nada que elegir. El entorno marcado se exporta como `GENIX_CONFIG_FILE` a
todos los procesos hijos; en modo no interactivo se respeta el `GENIX_CONFIG_FILE` que ya
venga del entorno y, si no hay, se usa `config.toml`.

Cambiar de entorno **rearma la pestaña Actions**, porque el botón de backend depende del
`providers.backend` de ese archivo.

En Actions y Scripts, espacio o click alternan la marca y Enter ejecuta todo lo marcado en
**ambas** pestañas: primero las acciones, después los scripts. `q`/`Esc`/`Ctrl+C` cancelan.

El backend es **un solo botón**, elegido según el `providers.backend` del entorno seleccionado:

| `providers.backend` | Botón | Ejecuta |
|---|---|---|
| `aws` | `Backend (AWS Cloud)` | `cd cloud && go run . accion=1` |
| `none` o vacío | `Backend (VPS)` | `cd scripts && go run . deploy_vps` |
| `cloudflare` | `Backend (cloudflare no soportado)`, deshabilitado | — |

## Estados del botón

| Estado | Cómo se ve |
|---|---|
| Normal | Borde gris |
| Seleccionado | Borde verde y `✓` en la etiqueta |
| Enfocado | Sólo el borde inferior, grueso y azul |
| Deshabilitado | Gris tenue; ignora el click y el Enter |

Al arrancar **no hay nada enfocado**: el indicador de foco aparece recién con la primera tecla
de navegación y hacer click no lo activa, porque un botón resaltado se confunde con uno
seleccionado. Marcar y ejecutar con el mouse no requiere foco en ningún momento.

## Grilla

Todos los botones miden lo mismo. El número de columnas es el mayor que entre en la terminal
**sin que ninguna etiqueta haya que recortar**: las etiquetas largas se parten en dos líneas y,
si a un cierto número de columnas alguna no entra en dos líneas, se usa una columna menos. Por
eso una terminal de 145 columnas muestra 3 y una de 110 muestra 2.

## Scripts (reemplazo de app.sh)

La pestaña Scripts replica exactamente los comandos que despacha `app.sh`, y las claves son las
mismas, así que también funcionan por línea de comandos:

```bash
./deploy.sh check_tables
./deploy.sh edit product_inventory category:string
./deploy.sh 1 check_tables          # acciones y scripts en la misma corrida
```

| Grupo | Scripts |
|---|---|
| Base de Datos | `check_tables`, `create`, `edit` |
| Servidores | `configure`, `follow_cloudwatch_logs` |
| Generadores | `generate_controllers`, `sync_struct_interfaces`, `generate_menu_descriptions`, `generate_sale_orders` |

`create` y `edit` necesitan argumentos: si no vinieron por línea de comandos se piden por stdin
antes de ejecutar. Los tres `configure_*` preguntan su propio modo de instalación, así que no se
les pide nada acá.

`app.sh` sigue existiendo y funciona igual, pero ya no hace falta: todo lo que despachaba está
cubierto por `./deploy.sh`.

## Orden de ejecución

Las acciones **no** corren en el orden en que se marcaron, sino en el orden fijo
`14, 1, 8, 2, 3, 4, 9, 5, 6, 10, 7, 11, 13, 12` (constante `executionOrder`). La
infraestructura (9) va antes que las tablas porque CloudFormation es dueño de la tabla
DynamoDB que `fn-init` necesita. Marcar la 5 junto con la 6 no la ejecuta dos veces.

Si alguna de las acciones 1, 2, 3 o 4 está marcada se hace `git pull` antes; que falle no
aborta el despliegue.

## Archivos

| Archivo | Contenido |
|---|---|
| `button.go` | El componente `Button`, el cálculo de columnas, `layoutTabs` y `layoutSections`, que devuelven en una sola pasada las líneas dibujadas y la caja de click de cada elemento. Es lo que garantiza que el mouse no se desalinee del render. |
| `tui.go` | El modelo Bubble Tea: navegación entre las tres secciones. |
| `actions.go` | Registro de acciones, orden de ejecución y helpers de `exec`. |
| `scripts.go` | Registro de los utilitarios de `app.sh`. |
| `lambda_env.go` | Acción 13: comprime las credenciales (zstd + base64 url-safe) en Go y las inyecta con `aws lambda update-function-configuration`. Ya no hacen falta `jq`, `zstd` ni `base64`. |
| `main.go` | Resuelve la raíz del repo, el modo (argumentos vs interfaz) y el binario de Go. |

## Agregar una acción o un script

Para una acción: agregar una entrada a `deployActions` (en `actions.go`) con su `id`, `group`,
`label` y `run`, y agregar el `id` a `executionOrder` en la posición que corresponda. El test
`TestEveryActionIsExecutable` falla si se olvida lo segundo.

Para un script: agregar una entrada a `deployScripts` (en `scripts.go`) con su `key`, `group`,
`label` y `run`. Si necesita argumentos, poner `argumentsHint` y se piden solos.

## Tests

```bash
cd scripts && go test ./deployer
```

Cubren el calce entre las cajas del mouse y los bordes dibujados en varias anchuras, que la
grilla sea uniforme y no recorte etiquetas entre 40 y 200 columnas, que sin foco no se dibuje
el indicador, el parseo de argumentos, el colapso del botón de backend por `providers.backend`
y que toda acción del menú sea alcanzable.
