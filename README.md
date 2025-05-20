# keeper-chat
chatroom for the keeper project

## Running with Docker Compose

This project includes a Docker Compose setup to easily run the Go backend server and the Flutter test UI (web build) in containers.

### Prerequisites

-   [Docker](https://docs.docker.com/get-docker/) installed.
-   [Docker Compose](https://docs.docker.com/compose/install/) installed (often included with Docker Desktop).

### Usage

1.  **Build and Run:**
    Open your terminal in the root directory of the project and run:
    ```bash
    docker-compose up --build
    ```
    This command will:
    -   Build the Docker image for the Go server (if not already built or if changes are detected).
    -   Build the Docker image for the Flutter test UI (web version).
    -   Start both containers.

2.  **Accessing the Services:**
    -   **Go Server API:** The server will be accessible at `http://localhost:8080`. You can test API endpoints like `/api/register` and `/api/login` using tools like `curl` or Postman.
    -   **Flutter Test UI:** The Flutter web application will be accessible at `http://localhost:8081`.

3.  **Stopping the Services:**
    To stop the services, press `Ctrl+C` in the terminal where `docker-compose up` is running.
    To stop and remove the containers, you can run:
    ```bash
    docker-compose down
    ```

### Data Persistence

The Go server uses an SQLite database. The database file (`keeper.db`) is stored in a Docker named volume (`keeper_data`) to persist data across container restarts.

### Development Notes for Flutter UI in Docker

-   The Flutter UI container serves a web build (`flutter build web`).
-   For active Flutter development, features like hot reload are best experienced by running the Flutter app directly on your host machine (e.g., `cd test-ui && flutter run`) and connecting to the Go server running in Docker (accessible at `http://localhost:8080`). The Docker setup is more for a stable, containerized version of the UI.
