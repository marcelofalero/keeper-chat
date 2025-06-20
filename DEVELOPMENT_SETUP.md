# Development Environment Setup

This guide provides detailed instructions for setting up your local development environment to work on this project, which utilizes Docker and various Ory services.

## Prerequisites

Ensure your system meets the following requirements:

-   **Docker**: Latest stable version. (Download from [Docker's Official Website](https://www.docker.com/products/docker-desktop))
-   **Docker Compose**: Typically included with Docker Desktop. If not, follow the [official installation guide](https://docs.docker.com/compose/install/).
-   **Make**:
    -   Linux: Usually pre-installed or available via your package manager (e.g., `sudo apt-get install make`).
    -   macOS: Pre-installed with Command Line Tools (run `xcode-select --install`).
    -   Windows: Can be installed via Chocolatey (`choco install make`), Cygwin, MinGW, or by using WSL (Windows Subsystem for Linux).
-   **Python**: Version 3.7+ (for the `initialize_hydra_secrets.py` script). (Download from [Python's Official Website](https://www.python.org/downloads/))
-   **Git**: For cloning the repository. (Download from [Git's Official Website](https://git-scm.com/downloads))

## 1. Clone the Repository

If you haven't already, clone the project repository to your local machine:

```bash
git clone <repository_url>
cd <repository_name>
```
Replace `<repository_url>` and `<repository_name>` with the actual URL and directory name.

## 2. Initialize Secrets

Before starting the services for the first time, you need to generate cryptographic secrets for Hydra, Kratos, and a password for the PostgreSQL database. A Python script is provided for this:

```bash
python initialize_hydra_secrets.py
```
This script will:
-   Create/update `config/hydra/hydra.yml` with generated secrets.
-   Create/update `config/kratos/kratos.yml` with generated secrets.
-   Create a `.env` file in the project root, containing the `POSTGRES_PASSWORD`. This `.env` file is used by Docker Compose to inject the password into the services.

**Important**: If `config/hydra/hydra.yml` or `config/kratos/kratos.yml` do not exist, the script will create them using placeholder templates and then populate the secrets.

## 3. Build Docker Images (Optional)

The `docker-compose.yml` file is configured to use pre-built images for Ory services from Docker Hub. However, if you have made changes to local Dockerfiles (e.g., for the `server` or `ui` services, if they exist and are built from source), you'll need to build them:

```bash
make build
# or
docker-compose build
```
If you are only using official images for all services, this step can be skipped as Docker Compose will pull them automatically.

## 4. Start the Environment

Use the Makefile to bring up all services:

```bash
make up
```
This command starts all services defined in `docker-compose.yml` in detached mode (`-d`). This includes:
-   `postgresd`: PostgreSQL database.
-   `hydra`, `kratos`, `keto`, `oathkeeper`: Ory services.
-   `hydra-migrate`, `kratos-migrate`, `keto-migrate`: Services that run database migrations.
-   `mailslurper`: Catches outgoing emails for development.
-   `server`: Your application server (if defined).
-   `ui`: Your UI application (if defined).

The first time you run this, Docker Compose will download any images not present locally.

After running `make up`, it's a good practice to ensure all database migrations have run successfully, especially for the first setup:
```bash
make init
```
This command specifically targets the Kratos and Keto migration processes.

## 5. Accessing Services and Logs

-   **Service Ports**: Refer to the "Accessing Services" section in `OPERATIONS.md` for a list of default ports for each service (UI, Kratos, Hydra, Keto, Oathkeeper, MailSlurper).
-   **Viewing Logs**: To view the logs for all running services:
    ```bash
    docker-compose logs -f
    ```
    To view logs for a specific service (e.g., `kratos`):
    ```bash
    docker-compose logs -f kratos
    ```
    This is crucial for debugging and understanding how services are interacting.

## 6. Developing Your Application

-   **Server (`server/`)**: If your application server is part of this Docker Compose setup (e.g., the Go server in `server/`), any changes you make to its source code might require a rebuild of its Docker image and a restart of the service:
    ```bash
    docker-compose build server
    docker-compose up -d --no-deps server
    ```
    (The `--no-deps` flag prevents restarting dependencies if they are already running).
-   **UI (`test-ui/`)**: Similar to the server, if your UI is containerized, changes might require a rebuild and restart.

## 7. Common Issues & Troubleshooting

-   **Permission Errors during Migrations**:
    -   If you see errors like `permission denied for schema public`, ensure that `scripts/init-db.sh` ran correctly and set the database ownership properly. This script is run automatically by the `postgresd` service on its first startup.
    -   You can try `make down`, then `make up` again to ensure `init-db.sh` is re-triggered if there were initial issues.
-   **Port Conflicts**:
    -   If any of the default ports (e.g., 8080, 4444, 4433) are already in use on your machine, Docker Compose will fail to start the respective service. You'll need to stop the conflicting service on your host or change the port mappings in `docker-compose.yml`.
-   **Incorrect Secrets**:
    -   If services behave unexpectedly regarding authentication or sessions, ensure `initialize_hydra_secrets.py` was run and that the `.env` file is present and sourced by Docker Compose.
-   **"Network not found" or "Service not found"**:
    -   Ensure Docker and Docker Compose are running correctly.
    -   Try `docker-compose down` and then `make up` for a fresh start.

## 8. Stopping the Environment

To stop all running services:

```bash
make down
```
This command also removes the containers and default networks but preserves database volumes by default.

For a more thorough cleanup including volumes and images, refer to `make pristine` in `OPERATIONS.md`.

---

This setup should provide a robust local environment for developing and testing the full authentication and authorization flow. For more details on operating the deployed services, see `OPERATIONS.md`.
