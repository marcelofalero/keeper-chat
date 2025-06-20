#!/usr/bin/env python3

import os
import secrets
import re
import yaml # Will need to ensure this is available or handle its absence

# Define Kubernetes Secret File Paths
K8S_HYDRA_SECRETS_PATH = "kubernetes/hydra-secrets.yml"
K8S_KRATOS_SECRETS_PATH = "kubernetes/kratos-secrets.yml"
K8S_POSTGRES_SECRETS_PATH = "kubernetes/postgres-secrets.yml"

# Base template for K8S Hydra secrets if not found
K8S_HYDRA_SECRETS_TEMPLATE = """apiVersion: v1
kind: Secret
metadata:
  name: hydra-secrets
type: Opaque
stringData:
  HYDRA_SECRETS_SYSTEM: "youReallyNeedToChangeThis"
  HYDRA_SECRETS_COOKIE: "youReallyNeedToChangeThisCookie"
  HYDRA_DSN: "postgres://hydra:{DB_PASSWORD}@postgres-svc:5432/hydra?sslmode=disable&max_conns=20&max_idle_conns=4"
  HYDRA_PAIRWISE_SALT: "youReallyNeedToChangeThisToo"
"""

# Base template for K8S Kratos secrets if not found
K8S_KRATOS_SECRETS_TEMPLATE = """apiVersion: v1
kind: Secret
metadata:
  name: kratos-secrets
type: Opaque
stringData:
  KRATOS_SECRETS_COOKIE: "kratosCookieSecretChangeMe"
  KRATOS_SECRETS_CIPHER: "kratosCipherSecretChangeMe"
  KRATOS_DSN: "postgres://kratos:{DB_PASSWORD}@postgres-svc:5432/kratos?sslmode=disable&max_conns=20&max_idle_conns=4"
"""

# Base template for K8S Postgres secrets if not found
K8S_POSTGRES_SECRETS_TEMPLATE = """apiVersion: v1
kind: Secret
metadata:
  name: postgres-secrets
type: Opaque
stringData:
  POSTGRES_USER: "postgres"
  POSTGRES_PASSWORD: "supersecretpassword"
  POSTGRES_DB: "defaultdb"
  POSTGRES_HOST: "postgres-svc"
  # For local port-forwarding or direct access if needed
  POSTGRES_PORT: "5432"
  # ROOT_PASSWORD is often the same as POSTGRES_PASSWORD for simplified setups
  POSTGRES_ROOT_PASSWORD: "supersecretpassword"
"""

def generate_hex_secret(length_bytes: int) -> str:
    """Generates a hex-encoded random string of length_bytes."""
    return secrets.token_hex(length_bytes)

def update_k8s_secret_file(file_path: str, secret_updates: dict, file_description: str, default_template: str, postgres_password_for_dsn: str = None) -> None:
    """
    Reads, updates, and writes a Kubernetes secret YAML file with new secret values.
    Updates are performed in the 'stringData' section.
    """
    try:
        with open(file_path, 'r') as f:
            content = f.read()
        yaml_data = yaml.safe_load(content)
        if not isinstance(yaml_data, dict): # Basic check if file is not empty/corrupt
            raise yaml.YAMLError("File content is not valid YAML")
        print(f"Successfully read {file_description} from {file_path}")
    except FileNotFoundError:
        print(f"Warning: {file_description} not found at {file_path}. Creating from template.")
        # Ensure directory exists
        parent_dir = os.path.dirname(file_path)
        if parent_dir and not os.path.exists(parent_dir):
            os.makedirs(parent_dir)
            print(f"Created directory: {parent_dir}")

        # Replace {DB_PASSWORD} in template if postgres_password_for_dsn is provided
        template_content = default_template
        if postgres_password_for_dsn:
            template_content = template_content.replace("{DB_PASSWORD}", postgres_password_for_dsn)

        with open(file_path, 'w') as f:
            f.write(template_content)
        print(f"Created a new {file_description} at {file_path} using the default template.")
        with open(file_path, 'r') as f:
            content = f.read()
        yaml_data = yaml.safe_load(content) # Load the newly created file
    except yaml.YAMLError as e:
        print(f"Error: Could not parse YAML in {file_path}. {e}")
        print("Please ensure the file is a valid YAML and Opaque Kubernetes Secret with a 'stringData' section.")
        # Try to create from template if parsing fails significantly
        if input(f"Do you want to overwrite {file_path} with a default template? (yes/no): ").lower() == 'yes':
            parent_dir = os.path.dirname(file_path)
            if parent_dir and not os.path.exists(parent_dir):
                os.makedirs(parent_dir)
            template_content = default_template
            if postgres_password_for_dsn:
                template_content = template_content.replace("{DB_PASSWORD}", postgres_password_for_dsn)
            with open(file_path, 'w') as f:
                f.write(template_content)
            print(f"Overwrote {file_path} with the default template.")
            return # Exit after overwrite to avoid further processing on potentially bad data
        else:
            print("Skipping update for this file due to parsing error and user choice not to overwrite.")
            return


    if 'stringData' not in yaml_data or not isinstance(yaml_data['stringData'], dict):
        print(f"Error: 'stringData' section is missing or not a dictionary in {file_path}.")
        # Offer to create from template if stringData is missing
        if input(f"Do you want to attempt to fix by creating from template (this will overwrite existing data)? (yes/no): ").lower() == 'yes':
            parent_dir = os.path.dirname(file_path)
            if parent_dir and not os.path.exists(parent_dir):
                os.makedirs(parent_dir)
            template_content = default_template
            if postgres_password_for_dsn:
                template_content = template_content.replace("{DB_PASSWORD}", postgres_password_for_dsn)
            with open(file_path, 'w') as f:
                f.write(template_content)
            print(f"Overwrote {file_path} with the default template and re-reading.")
            with open(file_path, 'r') as f: # Re-read after overwrite
                content = f.read()
            yaml_data = yaml.safe_load(content)
            if 'stringData' not in yaml_data or not isinstance(yaml_data['stringData'], dict):
                 print(f"Error: 'stringData' still missing after attempting template fix. Skipping {file_path}")
                 return # Skip if still not fixed
        else:
            print(f"Skipping update for {file_path} as 'stringData' is missing/invalid and user chose not to overwrite.")
            return

    updated_any_secret = False
    for yaml_key, new_value in secret_updates.items():
        if yaml_key in yaml_data['stringData']:
            if yaml_data['stringData'][yaml_key] != new_value:
                yaml_data['stringData'][yaml_key] = new_value
                print(f"  - Updated {yaml_key} in {file_description}.")
                updated_any_secret = True
            # else:
            #     print(f"  - {yaml_key} in {file_description} is already up to date.")
        else:
            # This case should ideally not happen if templates are correct
            # For DSN, we might add it if missing and password is provided
            if "DSN" in yaml_key and postgres_password_for_dsn:
                 yaml_data['stringData'][yaml_key] = new_value # The new_value here is the DSN string
                 print(f"  - Added {yaml_key} to {file_description} (was missing).")
                 updated_any_secret = True
            else:
                 print(f"  - Warning: Key '{yaml_key}' not found in 'stringData' of {file_description}. It will not be updated.")


    # Special handling for DSN fields if postgres_password_for_dsn is available
    # This ensures DSNs are updated if the password changes, even if not explicitly in secret_updates
    if postgres_password_for_dsn:
        if file_description == "Hydra Kubernetes Secrets":
            expected_dsn = f"postgres://hydra:{postgres_password_for_dsn}@postgres-svc:5432/hydra?sslmode=disable&max_conns=20&max_idle_conns=4"
            if yaml_data['stringData'].get("HYDRA_DSN") != expected_dsn:
                yaml_data['stringData']["HYDRA_DSN"] = expected_dsn
                print(f"  - Updated HYDRA_DSN in {file_description} due to new PostgreSQL password.")
                updated_any_secret = True
        elif file_description == "Kratos Kubernetes Secrets":
            expected_dsn = f"postgres://kratos:{postgres_password_for_dsn}@postgres-svc:5432/kratos?sslmode=disable&max_conns=20&max_idle_conns=4"
            if yaml_data['stringData'].get("KRATOS_DSN") != expected_dsn:
                yaml_data['stringData']["KRATOS_DSN"] = expected_dsn
                print(f"  - Updated KRATOS_DSN in {file_description} due to new PostgreSQL password.")
                updated_any_secret = True

    if updated_any_secret:
        try:
            with open(file_path, 'w') as f:
                yaml.dump(yaml_data, f, sort_keys=False, indent=2) # sort_keys=False to preserve order
            print(f"Successfully updated secrets in {file_path}")
        except Exception as e:
            print(f"Error writing updated YAML to {file_path}: {e}")
    else:
        placeholders_exist = False
        for k,v in yaml_data.get('stringData', {}).items():
            if isinstance(v, str) and ("youReallyNeedToChangeThis" in v or \
                                       "kratosCookieSecretChangeMe" in v or \
                                       "kratosCipherSecretChangeMe" in v or \
                                       "supersecretpassword" in v or \
                                       "{DB_PASSWORD}" in v) and \
                                       k in secret_updates: # Check if this placeholder is one we manage
                placeholders_exist = True
                break
        if placeholders_exist:
            print(f"No specific values were updated in {file_path}, but placeholders still exist for managed keys. This might indicate an issue or that placeholders are different than expected.")
        else:
            print(f"Secrets in {file_path} appear to be already up-to-date and no standard placeholders found for managed keys.")


def main():
    print("Initializing Kubernetes secrets...")

    # Generate secrets
    # For Hydra, only one system secret is used in the k8s YAML.
    hydra_system_secret = generate_hex_secret(32)
    hydra_cookie_secret = generate_hex_secret(32)
    hydra_salt_secret = generate_hex_secret(32) # For pairwise subject identifiers

    kratos_cookie_secret = generate_hex_secret(32)
    kratos_cipher_secret = generate_hex_secret(16) # Kratos uses 16 or 32 bytes for AES, 16 bytes = 32 hex chars

    postgres_password = generate_hex_secret(16) # 32 hex chars

    # Prepare updates for each file
    hydra_k8s_updates = {
        "HYDRA_SECRETS_SYSTEM": hydra_system_secret,
        "HYDRA_SECRETS_COOKIE": hydra_cookie_secret,
        "HYDRA_PAIRWISE_SALT": hydra_salt_secret
        # HYDRA_DSN will be updated/set by update_k8s_secret_file using postgres_password
    }

    kratos_k8s_updates = {
        "KRATOS_SECRETS_COOKIE": kratos_cookie_secret,
        "KRATOS_SECRETS_CIPHER": kratos_cipher_secret
        # KRATOS_DSN will be updated/set by update_k8s_secret_file using postgres_password
    }

    postgres_k8s_updates = {
        "POSTGRES_PASSWORD": postgres_password,
        "POSTGRES_ROOT_PASSWORD": postgres_password # Usually same for simplicity in dev
    }

    # Ensure PyYAML is available
    try:
        import yaml
    except ImportError:
        print("Error: PyYAML is not installed. This script requires PyYAML to operate on Kubernetes secret files.")
        print("Please install it, e.g., 'pip install PyYAML'")
        # Attempt to run a bash command to install it (won't work if pip not there or no permission)
        try:
            print("Attempting to install PyYAML...")
            import subprocess
            subprocess.check_call(["pip", "install", "PyYAML"])
            print("PyYAML installed successfully. Please re-run the script.")
        except Exception as e:
            print(f"Could not auto-install PyYAML: {e}")
        return # Exit if PyYAML is not available

    # Update Kubernetes Secret Files
    # For Hydra and Kratos, pass postgres_password to correctly set their DSNs
    update_k8s_secret_file(K8S_HYDRA_SECRETS_PATH, hydra_k8s_updates, "Hydra Kubernetes Secrets", K8S_HYDRA_SECRETS_TEMPLATE, postgres_password_for_dsn=postgres_password)
    update_k8s_secret_file(K8S_KRATOS_SECRETS_PATH, kratos_k8s_updates, "Kratos Kubernetes Secrets", K8S_KRATOS_SECRETS_TEMPLATE, postgres_password_for_dsn=postgres_password)
    update_k8s_secret_file(K8S_POSTGRES_SECRETS_PATH, postgres_k8s_updates, "PostgreSQL Kubernetes Secrets", K8S_POSTGRES_SECRETS_TEMPLATE)


    try:
        os.chmod(__file__, 0o755) # Make script executable
        # print(f"Made {__file__} executable.") # Less verbose
    except Exception as e:
        print(f"Warning: Could not make {__file__} executable: {e}")

    print("Kubernetes secrets initialization process complete.")

if __name__ == "__main__":
    main()
