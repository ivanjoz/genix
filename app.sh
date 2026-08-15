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
    # For "configure_db", configure ScyllaDB and/or install the GenixSearch and Qdrant services.
    # Optional mode argument: all (default) = every engine, scylla, search or qdrant = only that one.
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
  "follow_cloudwatch_logs")
    # Follow the main backend Lambda log group from the selected config file.
    echo "Executing follow_cloudwatch_logs script..."
    (cd scripts && go run . follow_cloudwatch_logs)
    ;;
  "generate_sale_orders")
    # For "generate_sale_orders", run the backend sample-record generator.
    echo "Executing generate_sale_orders command..."
    (cd backend && go run . fn-generate-sale-orders)
    ;;
  "generate_erp_history")
    # For "generate_erp_history", replay past days of purchases, receptions and sales.
    echo "Executing generate_erp_history command..."
    (cd scripts && go run . generate_erp_history "${@:2}")
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
  "generate_route_ids")
    # For "generate_route_ids", assign a stable int16 to every ModuleHandlers route and rewrite
    # backend/core/api_routes.generated.go. Pass --check to fail instead of writing.
    echo "Executing generate_route_ids command..."
    (cd scripts && go run . generate_route_ids "${@:2}")
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
  "index_documentation")
    # Validate by default; pass -mode index to update Qdrant incrementally.
    echo "Executing documentation indexer..."
    (cd backend && go run ./agent/cmd/documentation-index "${@:2}")
    ;;
  "search_documentation")
    # Run one read-only hybrid documentation query or the Spanish examples.
    echo "Executing documentation search..."
    (cd backend && go run ./agent/cmd/documentation-search "${@:2}")
    ;;
  *)
    # If the command is not recognized, show an error and usage instructions.
    echo "Unknown command: $1"
    echo "Usage: $0 {check_tables|create|edit|configure_server|configure_db|configure_server_utils|follow_cloudwatch_logs|generate_sale_orders|sync_struct_interfaces|generate_controllers|generate_route_ids|generate_menu_descriptions|index_documentation|search_documentation|deploy}"
    exit 1
    ;;
esac
