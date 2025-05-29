# keeper-chat
chatroom for the keeper project

## Running with Docker Compose

This project includes a Docker Compose setup to easily run the Go backend server and the Flutter test UI (web build) in containers.

### Prerequisites

-   [Docker](https://docs.docker.com/get-docker/) installed.
-   [Docker Compose](https://docs.docker.com/compose/install/) installed (often included with Docker Desktop).
-   Python (for initializing Hydra secrets).

### Ory Hydra Setup (Authentication/Authorization)

This project now includes Ory Hydra for OAuth2/OIDC based authentication and authorization.
The Docker Compose setup includes the following services for Hydra:
-   `postgresd`: Hydra's PostgreSQL database.
-   `hydra-migrate`: A service that runs database migrations for Hydra.
-   `hydra`: The Ory Hydra OAuth2/OIDC server itself.
-   `consent`: An example login and consent UI application that interacts with Hydra.

**Crucially, before starting the services for the first time, or if you need to reset Hydra's secrets, you must run the initialization script:**
```bash
python initialize_hydra_secrets.py
```
This script will:
-   Generate necessary random secrets for Hydra.
-   Update `config/hydra/hydra.yml` with these secrets.
-   Create a `.env` file (e.g., containing `POSTGRES_PASSWORD`) which is used by `docker-compose.yml`.

You can interact with Hydra using its CLI through Docker Compose. For example, to create an OAuth2 client (after services are running):
```bash
docker-compose exec hydra hydra create client \
    --endpoint http://hydra:4445 \
    --grant-type client_credentials \
    --response-type token \
    --name "My Test Client" \
    --token-endpoint-auth-method client_secret_post
```
Replace placeholders and parameters as needed. The client ID and secret will be output by this command.

### Usage

1.  **Starting the Services:**
    Ensure you have run the `python initialize_hydra_secrets.py` script as described above if this is the first time or if secrets need regeneration.
    Open your terminal in the root directory of the project and run:
    ```bash
    docker-compose up -d --build
    ```
    This command will:
    -   Build the Docker images for all services (if not already built or if changes are detected).
    -   Start all containers in detached mode.

2.  **Accessing the Services:**
    -   **Go Server API:** The server will be accessible at `http://localhost:8080`.
    -   **Flutter Test UI:** The Flutter web application will be accessible at `http://localhost:8081`.
    -   **Ory Hydra Public API:** `http://localhost:4444`
    -   **Ory Hydra Admin API:** `http://localhost:4445` (typically not accessed directly by end-users but by backend services or CLI tools)
    -   **Example Consent App:** `http://localhost:3000` (useful for testing login/consent flows)

3.  **Stopping the Services:**
    To stop the services, run:
    To stop and remove the containers, you can run:
    ```bash
    docker-compose down
    ```

### Data Persistence

The Go server uses an SQLite database. The database file (`keeper.db`) is stored in a Docker named volume (`keeper_data`) to persist data across container restarts.
Ory Hydra uses a PostgreSQL database. Its data is stored in a Docker named volume (`postgres_data`) to persist data across container restarts.

### Development Notes for Flutter UI in Docker

-   The Flutter UI container serves a web build (`flutter build web`).
-   For active Flutter development, features like hot reload are best experienced by running the Flutter app directly on your host machine (e.g., `cd test-ui && flutter run`) and connecting to the Go server running in Docker (accessible at `http://localhost:8080`). The Docker setup is more for a stable, containerized version of the UI.

## Database Seeding

To populate the database with initial test data (sample users and messages), you can run the seed script.
Ensure the server is not running when you run the seed script to avoid potential database lock issues, or ensure the script handles it.

1.  Make sure the `./data/` directory exists at the project root (Docker Compose bind mount should create it on first run if mapped to a new local dir, but `go run` might not). The script attempts to create it.
2.  From the project root directory, run:
    ```bash
    go run server/cmd/seed/main.go
    ```
This will create/populate the `./data/keeper.db` file.
