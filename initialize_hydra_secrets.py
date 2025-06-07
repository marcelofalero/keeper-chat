#!/usr/bin/env python3

import os
import secrets
import re

# Define paths
HYDRA_CONFIG_PATH = "config/hydra/hydra.yml"
KRATOS_CONFIG_PATH = "config/kratos/kratos.yml"
DOTENV_PATH = ".env"

# Placeholder for kratos.yml if not found
KRATOS_PLACEHOLDER_CONTENT = r"""
version: v0.13.0
dsn: postgres://kratos:${POSTGRES_PASSWORD}@postgresd:5432/kratos?sslmode=disable&max_conns=20&max_idle_conns=4

serve:
  public:
    base_url: http://127.0.0.1:4433/
    port: 4433
    cors:
      enabled: true
      allowed_origins:
        - http://127.0.0.1:8081 # UI
        - http://localhost:8081
  admin:
    port: 4434

selfservice:
  default_browser_return_url: http://127.0.0.1:8081/
  allowed_return_urls:
    - http://127.0.0.1:8081

  methods:
    password:
      enabled: true

  flows:
    error:
      ui_url: http://127.0.0.1:3000/error # Example error UI
    settings:
      ui_url: http://127.0.0.1:3000/settings
      privileged_session_max_age: 15m
    recovery:
      enabled: true
      ui_url: http://127.0.0.1:3000/recovery
    verification:
      enabled: true
      ui_url: http://127.0.0.1:3000/verification
      after:
        default_browser_return_url: http://127.0.0.1:8081/dashboard # Redirect to UI dashboard
    logout:
      after:
        default_browser_return_url: http://127.0.0.1:8081/login
    login:
      ui_url: http://127.0.0.1:3000/login # Kratos UI / Ory Account Experience
      lifespan: 12h
    registration:
      lifespan: 10m
      ui_url: http://127.0.0.1:3000/registration # Kratos UI / Ory Account Experience
      after:
        password:
          hooks:
            - hook: session

session:
  lifespan: 12h
  cookie:
    same_site: Lax
    # name: ory_kratos_session
    # domain: example.org # Set this to your domain
    # path: /
    # http_only: true
    # secure: true # Set to true in production

secrets:
  cookie:
    - "KRATOS_COOKIE_SECRET_PLACEHOLDER"
  cipher:
    - "KRATOS_CIPHER_SECRET_PLACEHOLDER"

log:
  level: debug
  format: text

identity:
  default_schema_id: default
  schemas:
    - id: default
      url: file:///etc/config/kratos/identity.schema.json

courier:
  smtp:
    connection_uri: smtps://test:test@mailslurper:1025/?skip_ssl_verify=true&legacy_ssl=true
    from_address: no-reply@ory.sh
"""

HYDRA_PLACEHOLDER_CONTENT = r"""
secrets:
  system:
    - "CHANGE_THIS_TO_A_SECURE_RANDOM_STRING_1"
    - "CHANGE_THIS_TO_A_SECURE_RANDOM_STRING_2"
  cookie:
    - "CHANGE_THIS_TO_A_SECURE_RANDOM_COOKIE_SECRET"
oidc:
  subject_identifiers:
    pairwise:
      salt: "CHANGE_THIS_TO_A_SECURE_RANDOM_SALT_STRING"
"""

def generate_hex_secret(length_bytes: int) -> str:
    """Generates a hex-encoded random string of length_bytes."""
    return secrets.token_hex(length_bytes)

def update_config_file(config_path: str, new_secrets: dict, placeholder_content: str, replacements: list, config_name: str) -> None:
    """Reads, updates, and writes a configuration file with new secrets."""
    try:
        with open(config_path, 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print(f"Error: {config_name} configuration file not found at {config_path}")
        parent_dir = os.path.dirname(config_path)
        if parent_dir and not os.path.exists(parent_dir): # Check parent_dir is not empty string
            os.makedirs(parent_dir)
            print(f"Created directory: {parent_dir}")

        with open(config_path, 'w') as f:
            f.write(placeholder_content)
        print(f"Created a placeholder {config_name} configuration file at {config_path}")
        with open(config_path, 'r') as f:
            content = f.read()

    original_content = content
    updated_any_secret = False

    for key_name, placeholder, secret_key_in_dict in replacements:
        current_secret_value = new_secrets.get(secret_key_in_dict)
        if current_secret_value:
            # Pattern for list items: e.g., - "PLACEHOLDER"
            regex_list_item = rf'(\s*-\s*)"{re.escape(placeholder)}"'
            # Pattern for direct values: e.g., key: "PLACEHOLDER"
            regex_direct_value = rf'({re.escape(key_name)}:\s*)"{re.escape(placeholder)}"'

            new_content_attempt, num_replacements_attempt = content, 0

            # Try list item replacement first
            processed_content, num_replaced = re.subn(regex_list_item, rf'\1"{current_secret_value}"', content)
            if num_replaced > 0:
                new_content_attempt = processed_content
                num_replacements_attempt = num_replaced
            else:
                # Try direct value replacement if list item failed
                processed_content, num_replaced = re.subn(regex_direct_value, rf'\1"{current_secret_value}"', content)
                if num_replaced > 0:
                    new_content_attempt = processed_content
                    num_replacements_attempt = num_replaced

            if num_replacements_attempt > 0:
                content = new_content_attempt
                print(f"  - {config_name} {key_name.replace('_', ' ')} updated (placeholder: {placeholder}).")
                updated_any_secret = True

    if not updated_any_secret and content == original_content:
        already_fully_updated = True
        placeholders_found = False
        for _, placeholder_check, secret_key_check in replacements:
            if placeholder_check in content:
                placeholders_found = True
            if new_secrets.get(secret_key_check) not in content and placeholder_check in content:
                already_fully_updated = False
                break

        if already_fully_updated and not placeholders_found:
             print(f"Secrets in {config_path} seem to be already up-to-date (no placeholders found).")
        elif not placeholders_found and not updated_any_secret :
             print(f"No placeholders found in {config_path}. Assuming secrets are already managed or file is different than expected.")
        else:
            print(f"Warning: No secrets were updated in {config_path}. Placeholders might be missing or already replaced differently.")
    elif updated_any_secret:
        with open(config_path, 'w') as f:
            f.write(content)
        print(f"Successfully updated secrets in {config_path}")

def create_or_update_dotenv(dotenv_path: str, postgres_password: str) -> None:
    """Creates or updates the .env file with the PostgreSQL password."""
    with open(dotenv_path, 'w') as f:
        f.write(f"POSTGRES_PASSWORD={postgres_password}\n") # Added newline
    print(f"Created/Updated {dotenv_path} with POSTGRES_PASSWORD.")

def main():
    print("Initializing Hydra and Kratos secrets...")

    hydra_secrets_values = {
        "hydra_system_secret_1_val": generate_hex_secret(32),
        "hydra_system_secret_2_val": generate_hex_secret(32),
        "hydra_cookie_secret_val": generate_hex_secret(32),
        "hydra_salt_secret_val": generate_hex_secret(32),
    }
    kratos_secrets_values = {
        "kratos_cookie_secret_val": generate_hex_secret(32),  # 64 chars
        "kratos_cipher_secret_val": generate_hex_secret(16),  # 32 chars
    }
    postgres_password = generate_hex_secret(16)

    hydra_replacements = [
        ("system", "CHANGE_THIS_TO_A_SECURE_RANDOM_STRING_1", "hydra_system_secret_1_val"),
        ("system", "CHANGE_THIS_TO_A_SECURE_RANDOM_STRING_2", "hydra_system_secret_2_val"),
        ("cookie", "CHANGE_THIS_TO_A_SECURE_RANDOM_COOKIE_SECRET", "hydra_cookie_secret_val"),
        ("salt", "CHANGE_THIS_TO_A_SECURE_RANDOM_SALT_STRING", "hydra_salt_secret_val"),
    ]

    kratos_replacements = [
        ("cookie", "KRATOS_COOKIE_SECRET_PLACEHOLDER", "kratos_cookie_secret_val"),
        ("cipher", "KRATOS_CIPHER_SECRET_PLACEHOLDER", "kratos_cipher_secret_val"),
    ]

    for path in [HYDRA_CONFIG_PATH, KRATOS_CONFIG_PATH]:
        config_dir = os.path.dirname(path)
        if config_dir and not os.path.exists(config_dir): # Ensure config_dir is not empty for root files
            os.makedirs(config_dir)
            print(f"Created directory: {config_dir}")

    update_config_file(HYDRA_CONFIG_PATH, hydra_secrets_values, HYDRA_PLACEHOLDER_CONTENT, hydra_replacements, "Hydra")
    update_config_file(KRATOS_CONFIG_PATH, kratos_secrets_values, KRATOS_PLACEHOLDER_CONTENT, kratos_replacements, "Kratos")
    create_or_update_dotenv(DOTENV_PATH, postgres_password)

    try:
        os.chmod(__file__, 0o755)
        print(f"Made {__file__} executable.")
    except Exception as e:
        print(f"Could not make {__file__} executable: {e}")

    print("Hydra and Kratos secrets initialization complete.")

if __name__ == "__main__":
    main()
