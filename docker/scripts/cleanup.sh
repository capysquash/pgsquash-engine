#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../scripts/lib/common.sh"

setup_error_handling

# Cleanup Script for pg-squash Docker Resources
# Removes validation containers, unused images, and temporary volumes

# Parse arguments
CLEAN_ALL=false
CLEAN_VALIDATION=true
CLEAN_VOLUMES=false
FORCE=false

show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Cleanup pg-squash Docker resources

OPTIONS:
    --all               Clean everything (containers, images, volumes)
    --validation-only   Clean only validation containers (default)
    --volumes           Also remove volumes
    --force             Skip confirmation prompts
    -h, --help          Show this help message

EXAMPLES:
    # Clean validation containers only
    $0

    # Clean everything including volumes
    $0 --all --volumes

    # Clean without confirmation
    $0 --all --force

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --all)
            CLEAN_ALL=true
            shift
            ;;
        --validation-only)
            CLEAN_VALIDATION=true
            shift
            ;;
        --volumes)
            CLEAN_VOLUMES=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            ;;
    esac
done

# Confirmation prompt
if [ "$FORCE" = false ]; then
    echo "🧹 This will clean up pg-squash Docker resources:"
    [ "$CLEAN_VALIDATION" = true ] && echo "  • Validation containers"
    [ "$CLEAN_ALL" = true ] && echo "  • All pg-squash containers and images"
    [ "$CLEAN_VOLUMES" = true ] && echo "  • Docker volumes"
    echo ""
    read -p "Continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Cancelled by user"
        exit 0
    fi
fi

log_info "🧹 Starting cleanup..."

# Clean validation containers
if [ "$CLEAN_VALIDATION" = true ]; then
    log_info "Removing validation containers..."

    # Stop and remove containers with pg-squash labels
    VALIDATION_CONTAINERS=$(docker ps -aq -f "label=pg-squash.type=validation" 2>/dev/null || true)
    if [ -n "$VALIDATION_CONTAINERS" ]; then
        echo "$VALIDATION_CONTAINERS" | xargs docker rm -f 2>/dev/null || true
        log_success "Removed validation containers"
    else
        log_info "No validation containers found"
    fi

    # Remove containers matching name pattern
    NAMED_CONTAINERS=$(docker ps -aq -f "name=pgsquash-validation-" 2>/dev/null || true)
    if [ -n "$NAMED_CONTAINERS" ]; then
        echo "$NAMED_CONTAINERS" | xargs docker rm -f 2>/dev/null || true
        log_success "Removed named validation containers"
    fi
fi

# Clean all pg-squash resources
if [ "$CLEAN_ALL" = true ]; then
    log_info "Removing all pg-squash containers..."

    # Stop all containers with pg-squash label
    ALL_CONTAINERS=$(docker ps -aq -f "label=pg-squash" 2>/dev/null || true)
    if [ -n "$ALL_CONTAINERS" ]; then
        echo "$ALL_CONTAINERS" | xargs docker rm -f 2>/dev/null || true
        log_success "Removed all pg-squash containers"
    fi

    # Stop containers from docker-compose
    for compose_file in docker/deployment/*.yml; do
        if [ -f "$compose_file" ]; then
            log_info "Stopping services in $(basename "$compose_file")..."
            docker compose -f "$compose_file" down 2>/dev/null || true
        fi
    done

    log_info "Removing pg-squash images..."
    docker images | grep "pgsquash" | awk '{print $3}' | xargs docker rmi -f 2>/dev/null || true
    log_success "Removed pg-squash images"
fi

# Clean volumes
if [ "$CLEAN_VOLUMES" = true ]; then
    log_info "Removing pg-squash volumes..."

    # Remove labeled volumes
    VOLUMES=$(docker volume ls -q -f "label=pg-squash" 2>/dev/null || true)
    if [ -n "$VOLUMES" ]; then
        echo "$VOLUMES" | xargs docker volume rm 2>/dev/null || true
        log_success "Removed pg-squash volumes"
    else
        log_info "No pg-squash volumes found"
    fi

    # Prune anonymous volumes
    log_info "Pruning anonymous volumes..."
    docker volume prune -f 2>/dev/null || true
fi

# System cleanup
log_info "Running Docker system prune..."
if [ "$FORCE" = true ]; then
    docker system prune -f 2>/dev/null || true
else
    docker system prune 2>/dev/null || true
fi

# Summary
log_success "✅ Cleanup completed!"
echo ""
log_info "📊 Current Docker usage:"
docker system df

echo ""
log_info "💡 Tips:"
log_info "  • Run '$0 --all' to clean everything"
log_info "  • Run '$0 --all --volumes' to also remove volumes"
log_info "  • Run 'docker ps -a' to see remaining containers"
log_info "  • Run 'docker images' to see remaining images"
