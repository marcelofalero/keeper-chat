#!/usr/bin/env python3

import os
import secrets
import re

# Define paths
HYDRA_CONFIG_PATH = "config/hydra/hydra.yml"
DOTENV_PATH = ".env"

def generate_hex_secret(length_bytes: int) -> str:
    """Generates a hex-encoded random string of length_bytes."""
    return secrets.token_hex(length_bytes)

def update_hydra_config(config_path: str, new_secrets: dict) -> None:
    """Reads, updates, and writes Hydra configuration with new secrets."""
    try:
        with open(config_path, 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print(f"Error: Hydra configuration file not found at {config_path}")
        # Create directory if it doesn't exist
        os.makedirs(os.path.dirname(config_path), exist_ok=True)
        # Create a dummy file to allow the script to proceed with placeholder values
        # This is not ideal but allows the script to run if the file is missing
        # and a subsequent step might populate it.
        # In a real scenario, this might be an error condition.
        with open(config_path, 'w') as f:
            f.write('''\
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
''')
        print(f"Created a placeholder Hydra configuration file at {config_path}")
        with open(config_path, 'r') as f:
            content = f.read()


    original_content = content

    # Update system secrets
    content = re.sub(
        r'(\s*-\s*)"CHANGE_THIS_TO_A_SECURE_RANDOM_STRING_1"',
        rf'\1"{new_secrets["system_secret_1"]}"',
        content
    )
    content = re.sub(
        r'(\s*-\s*)"CHANGE_THIS_TO_A_SECURE_RANDOM_STRING_2"',
        rf'\1"{new_secrets["system_secret_2"]}"',
        content
    )

    # Update cookie secret
    content = re.sub(
        r'(\s*-\s*)"CHANGE_THIS_TO_A_SECURE_RANDOM_COOKIE_SECRET"',
        rf'\1"{new_secrets["cookie_secret"]}"',
        content
    )

    # Update OIDC salt
    content = re.sub(
        r'(salt:\s*)"CHANGE_THIS_TO_A_SECURE_RANDOM_SALT_STRING"',
        rf'\1"{new_secrets["salt_secret"]}"',
        content
    )

    if content == original_content:
        print(f"Warning: No secrets were updated in {config_path}. Placeholders might be missing or already replaced.")
    else:
        with open(config_path, 'w') as f:
            f.write(content)
        print(f"Successfully updated secrets in {config_path}")
        if new_secrets.get("system_secret_1"):
            print(f"  - System secret 1 updated.")
        if new_secrets.get("system_secret_2"):
            print(f"  - System secret 2 updated.")
        if new_secrets.get("cookie_secret"):
            print(f"  - Cookie secret updated.")
        if new_secrets.get("salt_secret"):
            print(f"  - OIDC salt updated.")


def create_or_update_dotenv(dotenv_path: str, postgres_password: str) -> None:
    """Creates or updates the .env file with the PostgreSQL password."""
    with open(dotenv_path, 'w') as f:
        f.write(f"POSTGRES_PASSWORD={postgres_password}\n")
    print(f"Created/Updated {dotenv_path} with POSTGRES_PASSWORD.")
    print("Important: Ensure your docker-compose.yml uses this POSTGRES_PASSWORD for the 'postgresd' service.")
    print("You might need to update it from 'secret' to e.g., '${POSTGRES_PASSWORD}'.")

def main():
    print("Initializing Hydra secrets...")

    # Generate new secrets
    new_secrets = {
        "system_secret_1": generate_hex_secret(32),
        "system_secret_2": generate_hex_secret(32),
        "cookie_secret": generate_hex_secret(32),
        "salt_secret": generate_hex_secret(32),
    }
    postgres_password = generate_hex_secret(16)

    # Ensure config directory exists (though it should from previous steps)
    config_dir = os.path.dirname(HYDRA_CONFIG_PATH)
    if not os.path.exists(config_dir):
        os.makedirs(config_dir)
        print(f"Created directory: {config_dir}")

    # Update hydra.yml
    update_hydra_config(HYDRA_CONFIG_PATH, new_secrets)

    # Create/Update .env file
    create_or_update_dotenv(DOTENV_PATH, postgres_password)

    # Make the script executable (best effort, might not work in all environments)
    try:
        os.chmod(__file__, 0o755)
        print(f"Made {__file__} executable.")
    except Exception as e:
        print(f"Could not make {__file__} executable: {e}")

    print("Hydra secrets initialization complete.")

if __name__ == "__main__":
    main()
