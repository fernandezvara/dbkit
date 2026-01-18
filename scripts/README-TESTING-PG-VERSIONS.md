# PostgreSQL Testing Version Management

## Version Configuration

All PostgreSQL versions are centrally defined in `scripts/versions.mk`:

```makefile
# PostgreSQL versions configuration
export PG_VERSIONS="18:18.1 17:17.7 16:16.11 15:15.15 14:14.20 13:13.23"

# Extract version numbers for Makefile
export PG_18_VERSION="18.1"
export PG_17_VERSION="17.7"
export PG_16_VERSION="16.11"
export PG_15_VERSION="15.15"
export PG_14_VERSION="14.20"
export PG_13_VERSION="13.23"
```

## Generated Files

### docker-compose.yml

The `docker-compose.yml` file is generated from the version template using:

```bash
cd scripts
./generate-compose.sh
```

This creates a service for each PostgreSQL version with the correct image tag and port mapping.

### Makefile Integration

The Makefile includes the version variables and uses them in test targets:

```makefile
test-pg18:
	@echo "$(GREEN)Testing against PostgreSQL $(PG_18_VERSION)...$(NC)"
```
