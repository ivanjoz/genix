#!/bin/bash
# Wrapper del TUI de despliegue. Toda la lógica vive en scripts/deployer (Go + Bubble Tea).
# Sin argumentos abre la interfaz; con argumentos ejecuta las acciones directamente:
#   ./deploy.sh          -> interfaz de botones
#   ./deploy.sh 6 9      -> acciones 6 y 9
#   ./deploy.sh 11 42    -> acción 11 para el CompanyID 42
source ~/.bashrc 2> /dev/null

cd "$(dirname "$0")/scripts" || exit 1

GO_PATH="go"
if [ -x /usr/local/go/bin/go ]; then
    GO_PATH="/usr/local/go/bin/go"
fi

exec "$GO_PATH" run ./deployer "$@"
