#!/bin/bash

# This script acts as a router for other scripts.
# DEPRECATED: every command below is also available in ./deploy.sh, either from the "Scripts"
# tab of its TUI or by name (e.g. ./deploy.sh check_tables). See scripts/DEPLOYER.md.

# Check for the first argument to decide which script to run.
case "$1" in
  "check_tables")
    # For "check_tables", execute the Go script in the "scripts" folder.
    echo "Executing check_tables script..."
    (cd scripts && go run . check_tables)
    ;;
  "create")
    # For "create", create a new database table structure.
    echo "Executing create command..."
    (cd scripts && go run table/create_edit_table.go create "${@:2}")
    ;;
  "edit")
    # For "edit", add a new column to an existing table.
    echo "Executing edit command..."
    (cd scripts && go run table/create_edit_table.go edit "${@:2}")
    ;;
  "configure_server")
    # For "configure_server", install the systemd services and/or the Nginx reverse proxy.
    # Optional mode argument: 1 = full, 2 = only systemd service, 3 = only Nginx proxy.
    echo "Executing configure_server script..."
    python3 scripts/configure_server.py "${@:2}"
    ;;
  "configure_db")
    # For "configure_db", configure ScyllaDB and/or install the GenixSearch service on this host.
    # Optional mode argument: all (default) = both, scylla = only ScyllaDB, search = only GenixSearch.
    echo "Executing configure_db script..."
    python3 scripts/configure_db.py "${@:2}"
    ;;
  "configure_server_utils")
    # For "configure_server_utils", install the server_utils systemd service (credit rate limiter
    # + SSE bridge) plus the bridge's Nginx vhost on this host. No arguments: everything comes
    # from config.toml.
    echo "Executing configure_server_utils script..."
    python3 scripts/configure_server_utils.py "${@:2}"
    ;;
  "generate_sale_orders")
    # For "generate_sale_orders", run the backend sample-record generator.
    echo "Executing generate_sale_orders command..."
    (cd backend && go run . fn-generate-sale-orders)
    ;;
  "sync_struct_interfaces")
    # For "sync_struct_interfaces", align marked frontend interfaces with backend structs.
    echo "Executing sync_struct_interfaces command..."
    (cd scripts && go run . sync_struct_interfaces)
    ;;
  "generate_controllers")
    # For "generate_controllers", scan backend for scylla.TableStruct base structs and
    # rewrite backend/exec/controllers.generated.go.
    echo "Executing generate_controllers command..."
    (cd scripts && go run . generate_controllers)
    ;;
  "deploy")
    # For "deploy", open the deployment TUI (or run the given action IDs directly).
    (cd scripts && go run ./deployer "${@:2}")
    ;;
  "generate_menu_descriptions")
    # For "generate_menu_descriptions", export route markdown descriptions to tmp/menu_description.json.
    echo "Executing generate_menu_descriptions command..."
    (cd scripts && go run . generate_menu_descriptions)
    ;;
  *)
    # If the command is not recognized, show an error and usage instructions.
    echo "Unknown command: $1"
    echo "Usage: $0 {check_tables|create|edit|configure_server|configure_db|configure_server_utils|generate_sale_orders|sync_struct_interfaces|generate_controllers|generate_menu_descriptions|deploy}"
    exit 1
    ;;
esac
