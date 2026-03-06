#!/usr/bin/env bash
# setup-prerequisites.sh — Install prerequisites for golang-clean-architecture
# Compatible with: macOS, Linux (Debian/Ubuntu, RHEL/Fedora, Arch), Windows (WSL2)
set -euo pipefail

# ---------------------------------------------------------------------------
# Colors
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; }

# ---------------------------------------------------------------------------
# OS / Distro Detection
# ---------------------------------------------------------------------------
detect_os() {
    local uname_s
    uname_s="$(uname -s)"
    case "$uname_s" in
        Darwin)  OS="macos" ;;
        Linux)   OS="linux" ;;
        MINGW*|MSYS*|CYGWIN*)
            error "Native Windows is not supported. Please use WSL2."
            error "Install WSL2: wsl --install"
            exit 1
            ;;
        *)
            error "Unsupported OS: $uname_s"
            exit 1
            ;;
    esac
}

detect_distro() {
    DISTRO="unknown"
    if [ "$OS" = "linux" ]; then
        if [ -f /etc/os-release ]; then
            # shellcheck disable=SC1091
            . /etc/os-release
            case "$ID" in
                ubuntu|debian|pop|linuxmint|elementary|zorin)
                    DISTRO="debian" ;;
                fedora|rhel|centos|rocky|alma|ol)
                    DISTRO="rhel" ;;
                arch|manjaro|endeavouros)
                    DISTRO="arch" ;;
                *)
                    # Check for debian-based via ID_LIKE
                    if echo "${ID_LIKE:-}" | grep -q "debian"; then
                        DISTRO="debian"
                    elif echo "${ID_LIKE:-}" | grep -q "rhel\|fedora"; then
                        DISTRO="rhel"
                    else
                        warn "Unknown Linux distro: $ID. Will attempt Debian-style install."
                        DISTRO="debian"
                    fi
                    ;;
            esac
        else
            warn "Cannot detect Linux distro. Will attempt Debian-style install."
            DISTRO="debian"
        fi
    fi
}

# ---------------------------------------------------------------------------
# Tool Checks
# ---------------------------------------------------------------------------
check_command() {
    command -v "$1" &>/dev/null
}

check_go_version() {
    if ! check_command go; then
        return 1
    fi
    local ver
    ver="$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | sed 's/go//')"
    local major minor
    major="$(echo "$ver" | cut -d. -f1)"
    minor="$(echo "$ver" | cut -d. -f2)"
    if [ "$major" -gt 1 ] || { [ "$major" -eq 1 ] && [ "$minor" -ge 22 ]; }; then
        return 0
    fi
    return 1
}

check_docker_running() {
    docker info &>/dev/null 2>&1
}

# ---------------------------------------------------------------------------
# Installers — macOS (Homebrew)
# ---------------------------------------------------------------------------
install_macos() {
    # Ensure Homebrew is installed
    if ! check_command brew; then
        info "Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    fi

    local packages=()
    if ! check_go_version; then
        packages+=(go)
    fi
    if ! check_command docker; then
        info "Docker Desktop is required on macOS."
        info "Please install from: https://docs.docker.com/desktop/install/mac-install/"
        info "After installing, start Docker Desktop and re-run this script."
        if ! check_command docker; then
            warn "Docker not found. Continuing with other prerequisites..."
        fi
    fi
    if ! check_command psql; then
        packages+=(postgresql)
    fi
    if ! check_command make; then
        packages+=(make)
    fi
    if ! check_command git; then
        packages+=(git)
    fi

    if [ ${#packages[@]} -gt 0 ]; then
        info "Installing via Homebrew: ${packages[*]}"
        brew install "${packages[@]}"
    fi
}

# ---------------------------------------------------------------------------
# Installers — Debian/Ubuntu (apt)
# ---------------------------------------------------------------------------
install_debian() {
    info "Updating package list..."
    sudo apt-get update -qq

    local packages=()
    if ! check_go_version; then
        packages+=(golang)
    fi
    if ! check_command docker; then
        packages+=(docker.io)
    fi
    if ! check_command psql; then
        packages+=(postgresql-client)
    fi
    if ! check_command make; then
        packages+=(make)
    fi
    if ! check_command git; then
        packages+=(git)
    fi

    if [ ${#packages[@]} -gt 0 ]; then
        info "Installing via apt: ${packages[*]}"
        sudo apt-get install -y "${packages[@]}"
    fi

    # Docker Compose plugin
    if ! docker compose version &>/dev/null 2>&1; then
        info "Installing docker-compose-plugin..."
        sudo apt-get install -y docker-compose-plugin 2>/dev/null || {
            warn "docker-compose-plugin not in repos. Installing standalone docker-compose..."
            if ! check_command docker-compose; then
                sudo apt-get install -y docker-compose
            fi
        }
    fi

    # Add user to docker group
    if ! groups | grep -q docker; then
        info "Adding $USER to docker group..."
        sudo usermod -aG docker "$USER"
        warn "You may need to log out and back in for docker group to take effect."
    fi
}

# ---------------------------------------------------------------------------
# Installers — RHEL/Fedora (dnf)
# ---------------------------------------------------------------------------
install_rhel() {
    local packages=()
    if ! check_go_version; then
        packages+=(golang)
    fi
    if ! check_command docker; then
        packages+=(docker)
    fi
    if ! check_command psql; then
        packages+=(postgresql)
    fi
    if ! check_command make; then
        packages+=(make)
    fi
    if ! check_command git; then
        packages+=(git)
    fi

    if [ ${#packages[@]} -gt 0 ]; then
        info "Installing via dnf: ${packages[*]}"
        sudo dnf install -y "${packages[@]}"
    fi

    # Docker Compose plugin
    if ! docker compose version &>/dev/null 2>&1; then
        info "Installing docker-compose-plugin..."
        sudo dnf install -y docker-compose-plugin 2>/dev/null || {
            warn "docker-compose-plugin not available. Please install Docker Compose v2 manually."
        }
    fi

    # Add user to docker group
    if ! groups | grep -q docker; then
        info "Adding $USER to docker group..."
        sudo usermod -aG docker "$USER"
        warn "You may need to log out and back in for docker group to take effect."
    fi

    # Start Docker service
    if ! check_docker_running; then
        info "Starting Docker service..."
        sudo systemctl start docker
        sudo systemctl enable docker
    fi
}

# ---------------------------------------------------------------------------
# Installers — Arch (pacman)
# ---------------------------------------------------------------------------
install_arch() {
    local packages=()
    if ! check_go_version; then
        packages+=(go)
    fi
    if ! check_command docker; then
        packages+=(docker)
    fi
    if ! docker compose version &>/dev/null 2>&1; then
        packages+=(docker-compose)
    fi
    if ! check_command psql; then
        packages+=(postgresql-libs)
    fi
    if ! check_command make; then
        packages+=(make)
    fi
    if ! check_command git; then
        packages+=(git)
    fi

    if [ ${#packages[@]} -gt 0 ]; then
        info "Installing via pacman: ${packages[*]}"
        sudo pacman -S --noconfirm "${packages[@]}"
    fi

    # Add user to docker group
    if ! groups | grep -q docker; then
        info "Adding $USER to docker group..."
        sudo usermod -aG docker "$USER"
        warn "You may need to log out and back in for docker group to take effect."
    fi

    # Start Docker service
    if ! check_docker_running; then
        info "Starting Docker service..."
        sudo systemctl start docker
        sudo systemctl enable docker
    fi
}

# ---------------------------------------------------------------------------
# Check-only Mode
# ---------------------------------------------------------------------------
check_only() {
    local all_ok=true

    info "Checking prerequisites..."
    echo ""

    if check_go_version; then
        success "Go $(go version | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)"
    else
        error "Go >= 1.22 not found"
        all_ok=false
    fi

    if check_command docker; then
        success "Docker $(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
    else
        error "Docker not found"
        all_ok=false
    fi

    if docker compose version &>/dev/null 2>&1; then
        success "Docker Compose $(docker compose version --short 2>/dev/null || echo 'v2+')"
    elif check_command docker-compose; then
        success "docker-compose (standalone)"
    else
        error "Docker Compose not found"
        all_ok=false
    fi

    if check_command psql; then
        success "psql $(psql --version | grep -oE '[0-9]+\.[0-9]+' | head -1)"
    else
        error "psql (PostgreSQL client) not found"
        all_ok=false
    fi

    if check_command make; then
        success "make"
    else
        error "make not found"
        all_ok=false
    fi

    if check_command git; then
        success "git $(git --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"
    else
        error "git not found"
        all_ok=false
    fi

    if check_docker_running; then
        success "Docker daemon is running"
    else
        warn "Docker daemon is not running (start it before running setup)"
    fi

    echo ""
    if [ "$all_ok" = true ]; then
        success "All prerequisites are installed!"
        return 0
    else
        error "Some prerequisites are missing. Run this script without --check to install them."
        return 1
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    echo ""
    echo "========================================"
    echo "  Prerequisites Installer"
    echo "  golang-clean-architecture"
    echo "========================================"
    echo ""

    detect_os
    detect_distro
    info "Detected: OS=$OS, Distro=$DISTRO"
    echo ""

    # --check flag: only verify, don't install
    if [ "${1:-}" = "--check" ]; then
        check_only
        exit $?
    fi

    # Install based on platform
    case "$OS" in
        macos) install_macos ;;
        linux)
            case "$DISTRO" in
                debian) install_debian ;;
                rhel)   install_rhel ;;
                arch)   install_arch ;;
                *)      install_debian ;;  # fallback
            esac
            ;;
    esac

    echo ""
    info "Verifying installation..."
    check_only || {
        warn "Some tools may need manual installation. See messages above."
    }
}

main "$@"
