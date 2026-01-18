#!/bin/bash

# Generate docker-compose.yml from version template
set -e

cd "$(dirname "$0")"

# Read versions from versions.mk
source ./versions.mk

cat > ../docker-compose.yml << 'EOF'
services:
EOF

# Add each PostgreSQL version
for version in 18 17 16 15 14 13; do
    var_name="PG_${version}_VERSION"
    version_value="${!var_name}"
    
    cat >> ../docker-compose.yml << EOF
  postgres-${version}:
    image: docker.io/library/postgres:${version_value}
    container_name: dbkit-postgres-${version}
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: dbkit_test
    ports:
      - "54${version}:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

EOF
done

echo "Generated docker-compose.yml successfully!"
