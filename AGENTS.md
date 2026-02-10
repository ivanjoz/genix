# AGENT OPERATIONAL PROTOCOL (V2)

IMPORTANT: You are not an autonomous agent. The human is watching you very step and it has all the answers. Keep the human in the loop and ask questions.

IMPORTANT: If you have any questions, ask. NEVER have long trains of thought with yourself. Explan your rationale, you are pair programming.

## 1. INTELLIGENT RESEARCH
- **One-Shot Research:** Gather enough info in one search to form a hypothesis. Avoid "research loops" where you search for the same thing multiple times without writing code.

## 2. LOOP DETECTION & PREVENTION
- **The "Two-Strike" Rule:** If a tool call (shell command or file edit) returns the same error twice, or if the code state doesn't change after an application, you are STUCK.
- **STALL PROTOCOL:** Upon detecting a loop, you MUST stop all autonomous actions and present the following to the user:
    1. **Summary:** What was attempted and why it failed.
    2. **Hypothesis:** Your best guess on the root cause.
    3. **Assumption:** A specific assumption you are making to move forward.
    4. **The Ask:** "I'm stuck. Should I try [Proposed Fix] based on my assumption, or do you have a different direction?"

## 3. ITERATION STYLE
- **Keep in Loop:** Every tool execution should be preceded by a 1-sentence "Intent" (e.g., "Updating the ScyllaDB connection string to test the timeout hypothesis").
- **Extensive Logging**: Always implement and use debug logs extensively to diagnose errors and trace execution flow.

## 4. RULES
- Search for the correct .md documentation before proceed
- If some points in the task are unclear, stop and ask for clarification
- ALWAYS add concise comments in every code block to explain the rationale and the goal, especially when code contains business logic.
- ALWAYS use expresive names for varibles and functions. DONT USE generic names.

## Project Overview

The Genix project is an ERP and E-commerce platform for small businesses. It consists of a Go backend and a Svelte.js frontend. The project is currently migrating its frontend from Solid.js to Svelte.js.

## Backend

The backend is written in Go and uses ScyllaDB/Cassandra as its database. The backend code is located in the `backend/` directory.

## Key Documentation Files

### Project Overview
- **README.md** - General project overview: ERP+Ecommerce for small businesses

### Deployment
- **DEPLOYMENT.md** - Deployment options including AWS Lambda + ScyllaDB on VPS/EC2, self-host deployment with systemd

### Backend Documentation
- **backend/README.md** - Brief overview of the Go backend for Genix
- **backend/db/ORM_INTERNALS.md** - Deep dive into ORM internals: memory model, reflection engine, and query optimization
- **backend/docs/CREATE_API_HANDLERS.md** - API handler development guide, MUST read before creating APIs. Key concepts: "updated" parameter for delta responses, query examples, conventions.
- **backend/docs/ORM_DATABASE_QUERY.md** - Comprehensive ScyllaDB ORM documentation covering model definitions, CRUD operations, query building

### Frontend Documentation
- **frontend/FRONTEND.md** - Monorepo architecture with independent store app, directory structure, package system, development workflow
- **frontend/UI_COMPONENTS.md** - UI component library documentation: Page, OptionsStrip, Layer/Modal components, form components, VTable, services
- **frontend/STORE.md** - Store integration notes: thumbhash implementation, store routes, CSS hashing
- **frontend/docs/SERVICES_GUIDE.md** - Guide for creating frontend services (connectors), explaining Cached Services (Delta Cache) vs. Report Services. ALWAYS read before creating one.

### Scripts
- **scripts/CREATE_EDIT_TABLE.md** - Creates new database table structures and adds columns to existing tables. USE ALWAYS.
- **scripts/CHECK_TABLES_SCRIPT.md** - Validates data model conventions for the custom ORM
- **scripts/SCRIPTS.md** - Central dispatcher and wrapper script management for project utilities

### Frontend Rules
- Use untrack inside $effect to avoid render loop
- GetHandler fetched records need fields: "upd" (Updated) and "ID" (unique id) for delta cache. Or use GetHandler.keyID or .KeysIDs for setting another field.

### Backend Rules
- NEVER trust the client. ALWAYS validate the required field and consistency of the data, and return a descriptive error if any validation fails.
