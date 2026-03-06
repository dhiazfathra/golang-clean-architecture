#!/usr/bin/env bash
# setup.sh — Bootstrap infrastructure and start the server
# Usage: bash scripts/setup.sh [--skip-prereqs] [--no-seed] [--no-run] [--reset]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ---------------------------------------------------------------------------
# Colors
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; }
step()    { echo -e "\n${GREEN}▸ $*${NC}"; }

# ---------------------------------------------------------------------------
# Parse Flags
# ---------------------------------------------------------------------------
SKIP_PREREQS=false
NO_SEED=false
NO_RUN=false
RESET=false

for arg in "$@"; do
    case "$arg" in
        --skip-prereqs) SKIP_PREREQS=true ;;
        --no-seed)      NO_SEED=true ;;
        --no-run)       NO_RUN=true ;;
        --reset)        RESET=true ;;
        --help|-h)
            echo "Usage: bash scripts/setup.sh [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --skip-prereqs  Skip prerequisite check"
            echo "  --no-seed       Skip database seeding"
            echo "  --no-run        Setup infrastructure only, don't start server"
            echo "  --reset         Tear down existing infrastructure before starting"
            echo "  --help, -h      Show this help message"
            exit 0
            ;;
        *)
            error "Unknown flag: $arg"
            echo "Run with --help for usage."
            exit 1
            ;;
    esac
done

cd "$PROJECT_ROOT"

# ---------------------------------------------------------------------------
# Step 1: Check Prerequisites
# ---------------------------------------------------------------------------
if [ "$SKIP_PREREQS" = false ]; then
    step "Checking prerequisites..."
    if ! bash scripts/setup-prerequisites.sh --check; then
        echo ""
        error "Prerequisites missing. Install them with:"
        echo "  bash scripts/setup-prerequisites.sh"
        echo ""
        echo "Or re-run with --skip-prereqs to skip this check."
        exit 1
    fi
else
    info "Skipping prerequisite check (--skip-prereqs)"
fi

# ---------------------------------------------------------------------------
# Step 2: Environment File
# ---------------------------------------------------------------------------
step "Checking environment file..."
if [ ! -f .env ]; then
    info "Creating .env from .env.example..."
    cp .env.example .env
    warn "Review .env and update passwords before production use."
else
    success ".env already exists"
fi

# Source env vars for use in this script
set -a
# shellcheck disable=SC1091
source .env
set +a

# Ensure required vars have defaults
DATABASE_URL="${DATABASE_URL:-postgres://app:app@localhost:5432/app?sslmode=disable}"
VALKEY_URL="${VALKEY_URL:-localhost:6379}"

export DATABASE_URL VALKEY_URL

# ---------------------------------------------------------------------------
# Step 3: Reset (optional)
# ---------------------------------------------------------------------------
if [ "$RESET" = true ]; then
    step "Tearing down existing infrastructure..."
    make infra-down 2>/dev/null || true
    success "Infrastructure torn down"
fi

# ---------------------------------------------------------------------------
# Step 4: Start Infrastructure
# ---------------------------------------------------------------------------
step "Starting infrastructure (PostgreSQL + Valkey)..."
make infra-up
success "Docker containers started"

# ---------------------------------------------------------------------------
# Step 5: Wait for PostgreSQL
# ---------------------------------------------------------------------------
step "Waiting for PostgreSQL to be ready..."
MAX_RETRIES=30
RETRY_INTERVAL=1
for i in $(seq 1 $MAX_RETRIES); do
    if pg_isready -h localhost -p 5432 -U app &>/dev/null; then
        success "PostgreSQL is ready"
        break
    fi
    if [ "$i" -eq "$MAX_RETRIES" ]; then
        error "PostgreSQL did not become ready within ${MAX_RETRIES}s"
        error "Check: docker compose -f deployments/docker-compose.yaml logs postgres"
        exit 1
    fi
    printf "."
    sleep $RETRY_INTERVAL
done

# ---------------------------------------------------------------------------
# Step 6: Wait for Valkey
# ---------------------------------------------------------------------------
step "Waiting for Valkey to be ready..."
MAX_RETRIES=15
for i in $(seq 1 $MAX_RETRIES); do
    # Try connecting with valkey-cli or redis-cli, or netcat as fallback
    if command -v valkey-cli &>/dev/null && valkey-cli -h localhost -p 6379 ping &>/dev/null; then
        success "Valkey is ready"
        break
    elif command -v redis-cli &>/dev/null && redis-cli -h localhost -p 6379 ping &>/dev/null; then
        success "Valkey is ready (via redis-cli)"
        break
    elif nc -z localhost 6379 &>/dev/null 2>&1; then
        success "Valkey port is open"
        break
    fi
    if [ "$i" -eq "$MAX_RETRIES" ]; then
        error "Valkey did not become ready within ${MAX_RETRIES}s"
        error "Check: docker compose -f deployments/docker-compose.yaml logs valkey"
        exit 1
    fi
    printf "."
    sleep 1
done

# ---------------------------------------------------------------------------
# Step 7: Apply Migrations
# ---------------------------------------------------------------------------
step "Applying database migrations..."
make migrate
success "Migrations applied"

# ---------------------------------------------------------------------------
# Step 8: Seed Data
# ---------------------------------------------------------------------------
if [ "$NO_SEED" = false ]; then
    step "Seeding initial data..."
    make seed
    success "Seeding complete"
else
    info "Skipping seeding (--no-seed)"
fi

# ---------------------------------------------------------------------------
# Step 9: Summary
# ---------------------------------------------------------------------------
echo ""
echo "========================================"
echo -e "  ${GREEN}Setup Complete!${NC}"
echo "========================================"
echo ""
echo "  Server:    http://localhost${LISTEN_ADDR:-:8080}"
echo "  API Docs:  http://localhost${LISTEN_ADDR:-:8080}/docs"
echo "  Postgres:  localhost:5432 (user: app, db: app)"
echo "  Valkey:    localhost:6379"
echo ""
echo "  Default credentials (from seeders):"
echo "    super_admin / \$SEED_SUPER_ADMIN_PASSWORD"
echo ""

# ---------------------------------------------------------------------------
# Step 10: Start Server
# ---------------------------------------------------------------------------
if [ "$NO_RUN" = false ]; then
    step "Starting server..."
    make run
else
    info "Skipping server start (--no-run)"
    echo "  To start the server manually:"
    echo "    make run"
fi
