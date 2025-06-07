# Operation Manual

This document provides instructions for setting up, running, and interacting with the authentication and authorization system powered by Ory services (Hydra, Kratos, Keto, Oathkeeper).

## Prerequisites

Before you begin, ensure you have the following installed:
- Docker
- Docker Compose
- Make (usually available on Linux/macOS, Windows users might need to install it via Chocolatey or WSL)
- Python 3 (for the secret initialization script)

## First-Time Setup

1.  **Clone the Repository**:
    ```bash
    git clone <repository_url>
    cd <repository_name>
    ```

2.  **Initialize Secrets**:
    Run the Python script to generate necessary secrets for Hydra, Kratos, and the PostgreSQL database. This script creates/updates configuration files and a `.env` file.
    ```bash
    python initialize_hydra_secrets.py
    ```
    This will:
    - Generate random secrets for `config/hydra/hydra.yml`.
    - Generate random secrets for `config/kratos/kratos.yml`.
    - Create a `.env` file with a random `POSTGRES_PASSWORD`.

3.  **Start All Services**:
    Use the Makefile to bring up all services in detached mode. This will also run initial database migrations for Hydra, Kratos, and Keto.
    ```bash
    make up
    ```
    Wait for all services to start. You can check their status with `make status`.

4.  **Ensure Database Migrations (Optional but Recommended)**:
    Although `make up` should trigger migrations through the `xxx-migrate` services, you can explicitly ensure all migrations are applied by running:
    ```bash
    make init
    ```
    This command waits for PostgreSQL to be healthy and then runs the migration commands for Kratos and Keto directly.

## Common Commands

The `Makefile` provides several commands for managing the environment:

-   **`make up`**:
    Starts all Docker Compose services in detached mode. If services are already running, it will typically recreate containers that have configuration changes.

-   **`make down`**:
    Stops and removes all Docker Compose services, networks, and volumes defined in the `docker-compose.yml`. This is useful for a clean reset.

-   **`make status`**:
    Displays the current status of all services managed by Docker Compose (e.g., running, exited, ports).

-   **`make init`**:
    Initializes the system by running database migrations for Kratos and Keto. This should ideally be run after the first `make up`.

-   **`make help`**:
    Shows a list of available Make commands and their descriptions.

-   **`make pristine`**:
    Stops all services, removes all volumes, and removes all Docker images used by the services (as defined in `docker-compose.yml`). This is for a complete cleanup of the Docker environment. It does not affect your source files, `.env`, or configuration files in `config/`.

## Accessing Services

Here are the default ports for accessing the various services:

-   **Application UI**: `http://localhost:8081`
    -   This is your main web application.

-   **Ory Kratos (Public API)**: `http://localhost:4433`
    -   Kratos's public-facing API for login, registration, account management flows. Your UI will interact with this or be redirected to Kratos's own UI (if configured) via these endpoints.
-   **Ory Kratos (Admin API)**: `http://localhost:4434`
    -   Kratos's administrative API for managing identities, schemas, etc. (Typically not exposed publicly).

-   **Ory Hydra (Public API)**: `http://localhost:4444`
    -   Hydra's public-facing OAuth2.0 and OpenID Connect endpoints.
-   **Ory Hydra (Admin API)**: `http://localhost:4445`
    -   Hydra's administrative API for managing OAuth2 clients, consent flows, etc. (Typically not exposed publicly).

-   **Ory Keto (Read API)**: `http://localhost:4466`
    -   Keto's API for checking permissions.
-   **Ory Keto (Write API)**: `http://localhost:4467`
    -   Keto's API for managing permission rules (relationships). (Typically not exposed publicly).

-   **Ory Oathkeeper (Proxy)**: `http://localhost:4455`
    -   Oathkeeper's reverse proxy port. Requests to your protected services should go through this port.
-   **Ory Oathkeeper (API)**: `http://localhost:4456`
    -   Oathkeeper's API for managing rules, health checks, etc.

-   **MailSlurper (SMTP)**: `localhost:1025` (SMTP port)
-   **MailSlurper (Web UI)**: `http://localhost:4436`
    -   Catches emails sent by Kratos (e.g., for verification, recovery) for development purposes.

## User Registration

User registration is primarily handled by Ory Kratos.
-   If you are using a custom UI, it should make API calls to Kratos's registration flow endpoints (e.g., initialize flow, submit registration form).
-   The current Kratos configuration (`config/kratos/kratos.yml`) sets `selfservice.flows.registration.ui_url` to `http://127.0.0.1:8081/registration`. This means Kratos expects your UI application (running on port 8081) to provide the registration form at the `/registration` path.
-   Similarly for login, settings, recovery, etc., Kratos will redirect the browser to your UI application at the configured paths.

## Resetting the Environment

To completely reset the environment (e.g., remove all data, containers):
1.  Stop and remove containers, networks, and volumes:
    ```bash
    make down
    ```
    For a more thorough cleanup that also removes the Docker images used by the services, you can use:
    ```bash
    make pristine
    ```
    This command is more comprehensive than `make down -v`.

2.  (Optional) If you want to remove the generated secrets and `.env` file to start from scratch:
    ```bash
    rm .env
    # You might need to manually revert config/kratos/kratos.yml and config/hydra/hydra.yml to their placeholder states or re-run initialize_hydra_secrets.py
    ```
    A safer way to reset secrets is to just re-run `python initialize_hydra_secrets.py`.
3.  Then, you can go through the "First-Time Setup" steps again.

This provides a basic operational guide. Further details on specific Ory service interactions and advanced configurations can be found in the official Ory documentation.
