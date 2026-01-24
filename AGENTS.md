# Genix Project Structure for AI Agents

This document provides guidance for AI agents to navigate the Genix project codebase and find relevant documentation.

## Project Overview

The Genix project is an ERP and E-commerce platform for small businesses. It consists of a Go backend and a Svelte.js frontend. The project is currently migrating its frontend from Solid.js to Svelte.js.

For a general overview of the project, technology stack, and migration status, please refer to the main [README.md](README.md).

## Backend

The backend is written in Go and uses ScyllaDB/Cassandra as its database. The backend code is located in the `backend/` directory.

For detailed documentation on the backend, including the custom ORM for ScyllaDB/Cassandra, please refer to [backend/README.md](backend/README.md).

## Frontend

The project has two frontend directories:
- `frontend/`: The legacy frontend built with Solid.js.
- `frontend2/`: The new and current frontend built with Svelte.js and SvelteKit.

**All new frontend development should be done in `frontend2/`.**

The primary documentation for the frontend, including UI components, form handling, and data fetching, is in [frontend2/README.md](frontend2/README.md). This is the most important document for understanding the frontend architecture and conventions.

For historical context on the old Solid.js frontend, you can refer to [frontend/README.md](frontend/README.md).

## AI and Machine Learning

The `ai/` directory contains scripts and resources related to AI and machine learning models, including training and inference.

- For a quick start on using the AI functionalities, refer to [ai/QUICK_START.md](ai/QUICK_START.md).
- For details on the Gemma model function calling, see [ai/FUNCTION_GEMMA.md](ai/FUNCTION_GEMMA.md).
- For information on training the models, see [ai/README_TRAINING.md](ai/README_TRAINING.md).

## E-commerce Store

The `store/` directory contains the SvelteKit project for the e-commerce storefront. For basic information about this Svelte project, see [store/README.md](store/README.md).

## Deployment

Deployment instructions and scripts can be found in `DEPLOYMENT.md`, `deploy.sh`, and `backend/deploy.sh`.
