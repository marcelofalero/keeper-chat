# Operation Manual

This document provides instructions for setting up, running, and interacting with the authentication and authorization system powered by Ory services (Hydra, Kratos, Keto, Oathkeeper), primarily focusing on a Kubernetes (Minikube) environment.

## Kubernetes Operations (Minikube)

This section details how to manage and interact with the application stack deployed on Kubernetes via Minikube. Refer to the main `README.md` for initial setup and deployment instructions.

### Minikube Management

-   **Check Minikube Status:**
    ```bash
    minikube status
    ```

-   **Open Minikube Dashboard:**
    Provides a web UI to view and manage your Kubernetes cluster.
    ```bash
    minikube dashboard
    ```

-   **Accessing Services via Minikube:**
    While the `README.md` primarily describes using `kubectl port-forward` for accessing `ClusterIP` services, if you configure services with type `NodePort` or `LoadBalancer`, Minikube can provide direct access URLs:
    ```bash
    minikube service <service-name>
    ```
    For example, if `ui-service` was of type LoadBalancer: `minikube service ui-service`.

-   **Minikube Docker Environment (for Image Building):**
    Crucial for building local Docker images that Minikube can use. Ensure your terminal is configured to use Minikube's Docker daemon before building images.
    *   For Linux/macOS (bash/zsh):
        ```bash
        eval $(minikube -p minikube docker-env)
        ```
    *   For PowerShell:
        ```powershell
        minikube -p minikube docker-env | Invoke-Expression
        ```

### Managing Kubernetes Resources

-   **Apply Configurations:**
    Deploy or update resources using manifest files.
    ```bash
    kubectl apply -f <filename.yml>
    kubectl apply -f <directory_name>/
    ```
    Example: `kubectl apply -f kubernetes/`

-   **Delete Resources:**
    Remove resources from the cluster.
    ```bash
    kubectl delete -f <filename.yml>
    kubectl delete -f <directory_name>/
    ```
    Example: `kubectl delete -f kubernetes/`

### Inspecting Resources

-   **Viewing Pod Status:**
    List all pods in the current namespace, or watch for changes.
    ```bash
    kubectl get pods
    kubectl get pods -w
    ```

-   **Viewing Pod Logs:**
    Stream logs from a pod. If the pod has multiple containers, specify the container name.
    ```bash
    kubectl logs <pod-name>
    kubectl logs -f <pod-name> # Follow logs
    kubectl logs -c <container-name> <pod-name> # Specific container
    ```

-   **Getting Shell Access into a Pod:**
    Execute a command (like a shell) inside a running container.
    ```bash
    kubectl exec -it <pod-name> -- /bin/sh
    # or /bin/bash if available in the container
    # kubectl exec -it <pod-name> -c <container-name> -- /bin/sh # Specific container
    ```

-   **Describing Resources:**
    Get detailed information about a specific resource (e.g., its configuration, status, events).
    ```bash
    kubectl describe pod <pod-name>
    kubectl describe service <service-name>
    kubectl describe deployment <deployment-name>
    kubectl describe pvc <pvc-name>
    kubectl describe secret <secret-name>
    kubectl describe configmap <configmap-name>
    ```

-   **Checking Service Endpoints:**
    See the internal IP addresses and ports that a service is directing traffic to (the pods backing the service).
    ```bash
    kubectl get endpoints <service-name>
    ```

## User Registration

User registration is primarily handled by Ory Kratos.
-   If you are using a custom UI, it should make API calls to Kratos's registration flow endpoints.
-   The Kratos configuration (`kubernetes/kratos-configmap.yml` -> `kratos.yml`) sets `selfservice.flows.registration.ui_url` (e.g., `http://127.0.0.1:8081/registration`). This means Kratos expects your UI application (accessible via port-forwarding or an Ingress at the specified URL) to provide the registration form at the `/registration` path.
-   Similarly for login, settings, recovery, etc., Kratos will redirect the browser to your UI application at the configured paths. These URLs must be accessible to the end-user's browser.

---

## Archived: Docker Compose Operations (Outdated)

The following sections relate to the previous Docker Compose setup and are kept for archival purposes. **These commands are no longer applicable to the Kubernetes-based deployment.**

### Prerequisites (Docker Compose)

- Docker
- Docker Compose
- Make
- Python 3

### First-Time Setup (Docker Compose)

1.  **Clone Repository**
2.  **Initialize Secrets**: `python initialize_hydra_secrets.py`
3.  **Start All Services**: `make up`
4.  **Ensure Database Migrations**: `make init`

### Common Commands (Docker Compose Makefile)

-   **`make up`**: Starts services.
-   **`make down`**: Stops and removes services, networks, volumes.
-   **`make status`**: Displays service status.
-   **`make init`**: Runs database migrations.
-   **`make help`**: Shows Make commands.
-   **`make pristine`**: Full cleanup of Docker environment.

### Accessing Services (Docker Compose Defaults)

-   Application UI: `http://localhost:8081`
-   Ory Kratos (Public API): `http://localhost:4433`, (Admin API): `http://localhost:4434`
-   Ory Hydra (Public API): `http://localhost:4444`, (Admin API): `http://localhost:4445`
-   Ory Keto (Read API): `http://localhost:4466`, (Write API): `http://localhost:4467`
-   Ory Oathkeeper (Proxy): `http://localhost:4455`, (API): `http://localhost:4456`
-   MailSlurper: `localhost:1025` (SMTP), `http://localhost:4436` (Web UI)

### Resetting the Environment (Docker Compose)

1.  `make down` or `make pristine`.
2.  Optional: `rm .env` and re-run `python initialize_hydra_secrets.py`.
3.  Restart with "First-Time Setup".

---

This provides a basic operational guide. Further details on specific Ory service interactions and advanced configurations can be found in the official Ory documentation.
