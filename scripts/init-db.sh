#!/bin/bash
set -e

# Function to create a database if it doesn't exist
create_database() {
  local db_name=$1
  echo "  Creating database '$db_name' if it does not exist..."
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    SELECT 'CREATE DATABASE $db_name'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$db_name')\gexec
EOSQL
}

# Create databases for hydra, kratos, and keto
# The main "hydra" database is already created by the postgres entrypoint via POSTGRES_DB=hydra
# So we only need to explicitly create kratos and keto if they don't exist.

if [ -n "$KRATOS_DB_NAME" ]; then
  create_database "$KRATOS_DB_NAME"
fi

if [ -n "$KETO_DB_NAME" ]; then
  create_database "$KETO_DB_NAME"
fi

# Note: User creation and granting privileges might be necessary here
# if Kratos and Keto services don't have permissions to create tables
# in their respective databases using the main POSTGRES_USER.
# However, the DSNs are configured to use specific users (kratos, keto)
# which implies those users should be created with passwords.

# Example for creating users (if not handled by the services themselves using the main postgres user):
# CREATE USER kratos WITH PASSWORD 'your_kratos_password';
# GRANT ALL PRIVILEGES ON DATABASE kratos TO kratos;
# CREATE USER keto WITH PASSWORD 'your_keto_password';
# GRANT ALL PRIVILEGES ON DATABASE keto TO keto;

# For this setup, we assume the DSNs for Kratos and Keto are like:
# postgres://kratos_user:kratos_password@postgresd:5432/kratos_db
# postgres://keto_user:keto_password@postgresd:5432/keto_db
# And that these users (kratos_user, keto_user) need to be created, or that the main
# POSTGRES_USER has rights to create databases and the services will use that user
# to interact with their respective databases.

# The current configuration uses:
# DSN=postgres://kratos:${POSTGRES_PASSWORD}@postgresd:5432/kratos
# DSN=postgres://keto:${POSTGRES_PASSWORD}@postgresd:5432/keto
# This means Kratos and Keto will try to connect as users "kratos" and "keto" respectively,
# using the global POSTGRES_PASSWORD.
# We need to create these users if they don't exist and grant them privileges.

echo "  Creating user 'kratos' if it does not exist..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
  DO
  \$do\$
  BEGIN
    IF NOT EXISTS (
        SELECT FROM pg_catalog.pg_roles
        WHERE  rolname = 'kratos') THEN

        CREATE ROLE kratos LOGIN PASSWORD '$POSTGRES_PASSWORD';
    END IF;
  END
  \$do\$;
EOSQL

echo "  Creating user 'keto' if it does not exist..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
  DO
  \$do\$
  BEGIN
    IF NOT EXISTS (
        SELECT FROM pg_catalog.pg_roles
        WHERE  rolname = 'keto') THEN

        CREATE ROLE keto LOGIN PASSWORD '$POSTGRES_PASSWORD';
    END IF;
  END
  \$do\$;
EOSQL

# Grant privileges to the new users for their respective databases
if [ -n "$KRATOS_DB_NAME" ]; then
  echo "  Granting privileges on database '$KRATOS_DB_NAME' to user 'kratos'..."
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    GRANT ALL PRIVILEGES ON DATABASE $KRATOS_DB_NAME TO kratos;
EOSQL
fi

if [ -n "$KETO_DB_NAME" ]; then
  echo "  Granting privileges on database '$KETO_DB_NAME' to user 'keto'..."
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    GRANT ALL PRIVILEGES ON DATABASE $KETO_DB_NAME TO keto;
EOSQL
fi

echo "Database initialization script completed."
