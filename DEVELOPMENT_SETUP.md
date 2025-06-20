# Development Environment Setup

This guide provides detailed instructions for setting up your local development environment to work on this project, which utilizes Docker and various Ory services.

## Prerequisites

Ensure your system meets the following requirements:

-   **Docker**: Latest stable version. (Download from [Docker's Official Website](https://www.docker.com/products/docker-desktop))
-   **kubectl**: Latest stable version. (Install from [Kubernetes's Official Website](https://kubernetes.io/docs/tasks/tools/install-kubectl/))
-   **Local Kubernetes Cluster**: A local Kubernetes environment such as Minikube, Kind, or Docker Desktop Kubernetes. (Refer to their respective documentation for installation)
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

Before deploying services to Kubernetes, you need to generate cryptographic secrets for Hydra, Kratos, and a password for the PostgreSQL database. These secrets will be stored as Kubernetes secret manifest files. A Python script is provided for this:

```bash
python initialize_hydra_secrets.py
```
This script will generate or update the following Kubernetes secret files in the `kubernetes/` directory:
-   `kubernetes/hydra-secrets.yml`: Contains secrets for Ory Hydra.
-   `kubernetes/kratos-secrets.yml`: Contains secrets for Ory Kratos.
-   `kubernetes/postgres-secrets.yml`: Contains the PostgreSQL password.

**Important**: This script should be run *before* applying the Kubernetes manifests (e.g., via `kubectl apply -k kubernetes/`). This ensures that your Kubernetes cluster uses real, generated secrets rather than placeholder values that might be present in the template files. If the target secret files (e.g., `kubernetes/hydra-secrets.yml`) do not exist, the script will create them using base templates and then populate them with the generated secrets.

## 3. Building and Using Local Images

If you make changes to the Dockerfiles for local custom services (e.g., `server/` or `test-ui/`), you will need to rebuild the Docker images. The Ory stack images (Hydra, Kratos, etc.) are typically pulled from public registries and do not require local building unless you are customizing them directly.

Currently, the `Makefile` does not have specific helper targets for building individual local service images for Kubernetes deployment. You'll need to run `docker build` commands manually.

**1. Build the Image:**

Use the `docker build` command from the root of the repository. For example:

```bash
# For the server application (assuming Dockerfile is in ./server)
docker build -t keeper-server:latest ./server

# For the UI application (assuming Dockerfile is in ./test-ui)
docker build -t keeper-ui:latest ./test-ui
```
Replace `keeper-server:latest` and `keeper-ui:latest` with your desired image name and tag. The tag `latest` is common for local development.

**2. Load the Image into Your Local Kubernetes Cluster:**

After building the image, it needs to be available to your local Kubernetes cluster. The method depends on the tool you are using:

-   **Minikube**:
    ```bash
    minikube image load keeper-server:latest
    minikube image load keeper-ui:latest
    ```

-   **Kind**:
    ```bash
    # Replace <your_kind_cluster_name> with your cluster's name (often 'kind')
    kind load docker-image keeper-server:latest --name <your_kind_cluster_name>
    kind load docker-image keeper-ui:latest --name <your_kind_cluster_name>
    ```

-   **Docker Desktop Kubernetes**:
    Docker Desktop's built-in Kubernetes cluster typically shares Docker's image cache. If you build an image locally (e.g., `docker build -t my-image:custom .`), it should be automatically available to Kubernetes without an explicit "load" step, as long as the `imagePullPolicy` in your Kubernetes manifests allows it.

**3. ImagePullPolicy:**

The Kubernetes Deployment manifests for the custom services (e.g., `server-deployment.yml`, `ui-deployment.yml`) are configured with `imagePullPolicy: IfNotPresent`. This policy means that Kubernetes will first check if the image (e.g., `keeper-server:latest`) is already present in the cluster's local image cache. If it's found, the local image will be used. If not found, Kubernetes will attempt to pull it from a remote registry.

This setup is ideal for development, as it allows you to use locally built images simply by building them and (if necessary) loading them into your cluster's cache. Ensure the image name and tag in your `docker build` command match what's specified in your Kubernetes Deployment YAML files.

## 4. Deploying the Environment to Kubernetes

Once your secrets are initialized (Step 2) and any custom local images are built and loaded (Step 3), you can deploy the entire environment to your local Kubernetes cluster.

**1. Apply Kubernetes Manifests:**

Use the `kubectl apply` command, pointing it to the directory containing all your Kubernetes manifest files:

```bash
kubectl apply -f ./kubernetes/
```
This command instructs Kubernetes to create or update all the resources defined in the `.yml` files within the `kubernetes/` directory. This includes:
-   PostgreSQL database deployment and service.
-   Ory services (Kratos, Hydra, Keto, Oathkeeper) deployments and services.
-   Migration jobs for Kratos, Hydra, and Keto to set up their database schemas.
-   MailSlurper for email catching.
-   Your custom applications (e.g., `server`, `ui`) if their manifests are in this directory.
-   ConfigMaps for configuration and initialization scripts.
-   Secrets that you initialized earlier.

**2. Monitor Pod Startup:**

After applying the manifests, Kubernetes will begin pulling the necessary Docker images and starting the pods for each service. You can monitor the status of your pods:

```bash
kubectl get pods -w
```
Wait for the pods to reach `Running` status. For migration jobs (e.g., `kratos-migrate-xxx`, `hydra-migrate-xxx`), you should see them transition to `Completed` status. This indicates that the database schemas have been prepared.

**3. Database Initialization and Migrations:**

-   **Database Initialization**: The PostgreSQL database is automatically initialized when its pod starts. This process includes creating necessary users and databases as defined by initialization scripts (e.g., from `scripts/init-db.sh`, mounted via `kubernetes/init-db-configmap.yml`). This must complete before database migrations can run successfully.
-   **Schema Migrations**: Database schema migrations for Ory Kratos, Hydra, and Keto are handled by dedicated Kubernetes Deployments (e.g., `kratos-migrate-deployment.yml`, `hydra-migrate-deployment.yml`, `keto-migrate-deployment.yml`). These are applied as part of the `kubectl apply -f ./kubernetes/` command. The pods created by these deployments run the migration commands (e.g., `kratos migrate sql -e --yes`) and then complete. There is no separate `make init` command needed when using Kubernetes; the migrations are part of the deployment process.

Once all relevant pods are `Running` and migration jobs are `Completed`, the environment should be ready.

## 5. Accessing Services and Viewing Logs in Kubernetes

Once services are deployed and running in Kubernetes, you'll typically access them via `kubectl port-forward` for local development, as most services are exposed as `ClusterIP` type by default. Logs are viewed using `kubectl logs`.

**1. Accessing Services via Port Forwarding:**

You need to run each `kubectl port-forward` command in a separate terminal window, as it blocks while active.

-   **UI Service (Example: Nginx serving a static site like `test-ui`):**
    The UI service (`ui-service` defined in `kubernetes/ui-service.yml`) typically exposes port 80.
    ```bash
    # Access the UI at http://localhost:8081 (forwards local port 8081 to service port 80)
    kubectl port-forward service/ui-service 8081:80
    ```

-   **Kratos Public API:**
    The Kratos service (`kratos-public-service` or a similar name from `kubernetes/kratos-deployment.yml` which might define multiple ports) exposes its public API, typically on port 4433.
    ```bash
    # Access Kratos Public API at http://localhost:4433
    kubectl port-forward service/kratos-service 4433:4433
    ```
    *(Note: The actual service name for Kratos might be `kratos-service` or similar, check your `*-service.yml` files. The command above assumes `kratos-service` exposes both public and admin ports, or you might have separate services like `kratos-public-service` and `kratos-admin-service`)*

-   **Kratos Admin API:**
    The Kratos Admin API is typically on port 4434.
    ```bash
    # Access Kratos Admin API at http://localhost:4434
    kubectl port-forward service/kratos-service 4434:4434
    ```

-   **Other Services (Hydra, Keto, Oathkeeper, MailSlurper, PostgreSQL):**
    You can port-forward other services similarly.
    -   Find the service name and its `targetPort` or `port` from the respective `kubernetes/<service-name>-service.yml` file.
    -   Or use `kubectl get service` to list all services and their exposed ports.
    -   Example for MailSlurper (if service is named `mailslurper-service` exposing port 1080 for web UI and 1025 for SMTP):
        ```bash
        # Access MailSlurper Web UI at http://localhost:1080
        kubectl port-forward service/mailslurper-service 1080:1080
        # SMTP port (for applications to send email to)
        # kubectl port-forward service/mailslurper-service 1025:1025
        ```

**2. Viewing Logs:**

Use the `kubectl logs` command to view logs from specific pods or all pods matching a label.

-   **Get logs for a specific pod:**
    First, find the pod's full name using `kubectl get pods`. For example, if a Kratos pod is named `kratos-deployment-abcdef123-xyz789`:
    ```bash
    kubectl logs kratos-deployment-abcdef123-xyz789
    ```

-   **Follow logs for a deployment (using labels):**
    If your pods are consistently labeled (e.g., `app=kratos` in the Kratos deployment metadata), you can follow logs for all pods matching that label:
    ```bash
    kubectl logs -f -l app=kratos
    ```
    You can also follow logs for a specific deployment directly:
    ```bash
    kubectl logs -f deployment/kratos-deployment
    ```
    (Replace `kratos-deployment` with the actual name of the deployment).

---
**Advanced Operations:**

For more detailed information on interacting with your Kubernetes environment, including a wider range of `kubectl` commands for inspecting pods, services, managing resources, getting shell access to containers, and troubleshooting, **please refer to `OPERATIONS.MD`**. That document provides a more comprehensive guide to Kubernetes operations relevant to this project.
---

## 6. Developing Your Application (Server, UI, etc.)

When developing custom applications like the `server` (Go application in `server/`) or the `ui` (Flutter application in `test-ui/`), the typical workflow involves iterating on code changes and then deploying those changes to your local Kubernetes cluster.

The general steps are:

1.  **Make Code Changes**: Modify the source code of your application (e.g., in the `server/` or `test-ui/` directories).
2.  **Rebuild Docker Image**: After making changes, rebuild the Docker image for that specific application.
3.  **Ensure Image is Available to Cluster**: Load the newly built image into your local Kubernetes cluster's image cache.
4.  **Restart Kubernetes Deployment**: Trigger a rollout (restart) of the Kubernetes deployment for your application to pick up the new image.

**Detailed Steps:**

**A. Rebuild and Load Docker Image:**

Refer to **Section 3: Building and Using Local Images** for detailed instructions. As a reminder, you'll use `docker build` and then load the image into your cluster:

   -   **Build the image:**
       ```bash
       # For the server application (e.g., from ./server directory)
       docker build -t keeper-server:latest ./server

       # For the UI application (e.g., from ./test-ui directory)
       docker build -t keeper-ui:latest ./test-ui
       ```
       *(Ensure the tag, like `:latest`, matches what's in your Kubernetes deployment YAMLs for these services if you want `imagePullPolicy: IfNotPresent` to work seamlessly).*

   -   **Load the image into your cluster** (e.g., Minikube or Kind):
       ```bash
       # For Minikube
       minikube image load keeper-server:latest
       minikube image load keeper-ui:latest

       # For Kind (replace <your_kind_cluster_name> if not 'kind')
       kind load docker-image keeper-server:latest --name <your_kind_cluster_name>
       kind load docker-image keeper-ui:latest --name <your_kind_cluster_name>
       ```
       *(Docker Desktop Kubernetes might not require an explicit load step if the image is built in its Docker daemon).*

**B. Restart the Kubernetes Deployment:**

Once the new image is built and available in your cluster's cache, you need to tell Kubernetes to update the running application. You can do this by restarting the deployment:

```bash
# For the server application (assuming deployment is named 'server-deployment')
kubectl rollout restart deployment/server-deployment

# For the UI application (assuming deployment is named 'ui-deployment')
kubectl rollout restart deployment/ui-deployment
```
You can find the deployment names in your `kubernetes/<app>-deployment.yml` files or by running `kubectl get deployments`.

The `imagePullPolicy: IfNotPresent` (or `Never`, though `IfNotPresent` is more flexible) in your Deployment specifications is crucial here. It ensures that Kubernetes uses the local image (e.g., `keeper-server:latest`) if it finds it in its cache, rather than trying to pull it from an external registry. If you use a more specific tag (e.g., `keeper-server:v1.2.3`), you would update the image tag in the deployment YAML and re-apply it using `kubectl apply -f kubernetes/<app>-deployment.yml`. However, for rapid local development, using `:latest` and `rollout restart` is often more convenient.

After the rollout, Kubernetes will terminate the old pods and create new ones using the updated image. You can monitor this with `kubectl get pods -w`.

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
