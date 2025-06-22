#!/bin/bash

# Version management script for Flux Language
# This script helps with manual versioning and release preparation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to get current version
get_current_version() {
    local current_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    echo ${current_tag#v}
}

# Function to validate version format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        print_color $RED "Error: Version must be in format X.Y.Z (e.g., 1.2.3)"
        exit 1
    fi
}

# Function to check if version is greater than current
is_version_greater() {
    local current=$1
    local new=$2
    
    # Split versions into arrays
    IFS='.' read -ra CURRENT_PARTS <<< "$current"
    IFS='.' read -ra NEW_PARTS <<< "$new"
    
    # Compare major, minor, patch
    for i in 0 1 2; do
        if [ ${NEW_PARTS[i]} -gt ${CURRENT_PARTS[i]} ]; then
            return 0
        elif [ ${NEW_PARTS[i]} -lt ${CURRENT_PARTS[i]} ]; then
            return 1
        fi
    done
    
    return 1  # Versions are equal
}

# Function to suggest next version based on commit messages
suggest_version() {
    local current_version=$1
    local latest_tag="v${current_version}"
    
    # Get commits since last tag
    local commits=$(git rev-list ${latest_tag}..HEAD --oneline 2>/dev/null || git rev-list HEAD --oneline)
    
    if [ -z "$commits" ]; then
        print_color $YELLOW "No new commits since last tag."
        return
    fi
    
    # Split current version
    IFS='.' read -ra VERSION_PARTS <<< "$current_version"
    local major=${VERSION_PARTS[0]}
    local minor=${VERSION_PARTS[1]}
    local patch=${VERSION_PARTS[2]}
    
    # Analyze commits for conventional commit patterns
    local has_breaking=false
    local has_feature=false
    local has_fix=false
    
    while IFS= read -r commit; do
        if [[ "$commit" =~ (feat(\(.+\))?!:|BREAKING[[:space:]]CHANGE:|feat!:|fix!:|perf!:) ]]; then
            has_breaking=true
            break
        elif [[ "$commit" =~ feat(\(.+\))?:[[:space:]] ]]; then
            has_feature=true
        elif [[ "$commit" =~ (fix|perf)(\(.+\))?:[[:space:]] ]]; then
            has_fix=true
        fi
    done <<< "$commits"
    
    # Suggest version based on commit analysis
    if [ "$has_breaking" = true ]; then
        local suggested="$((major + 1)).0.0"
        print_color $RED "Breaking changes detected! Suggested version: ${suggested}"
    elif [ "$has_feature" = true ]; then
        local suggested="${major}.$((minor + 1)).0"
        print_color $GREEN "New features detected! Suggested version: ${suggested}"
    elif [ "$has_fix" = true ]; then
        local suggested="${major}.${minor}.$((patch + 1))"
        print_color $BLUE "Bug fixes detected! Suggested version: ${suggested}"
    else
        print_color $YELLOW "No conventional commits detected. Consider patch version: ${major}.${minor}.$((patch + 1))"
    fi
    
    echo
    print_color $YELLOW "Recent commits:"
    echo "$commits" | head -10
}

# Function to create a new version
create_version() {
    local version=$1
    validate_version "$version"
    
    local current_version=$(get_current_version)
    
    if ! is_version_greater "$current_version" "$version"; then
        print_color $RED "Error: New version (${version}) must be greater than current version (${current_version})"
        exit 1
    fi
    
    # Confirm with user
    print_color $YELLOW "Current version: ${current_version}"
    print_color $GREEN "New version: ${version}"
    echo -n "Create new version? (y/N): "
    read -r confirm
    
    if [[ ! $confirm =~ ^[Yy]$ ]]; then
        print_color $YELLOW "Version creation cancelled."
        exit 0
    fi
    
    # Create and push tag
    local tag="v${version}"
    git tag -a "$tag" -m "Release version ${version}"
    
    print_color $GREEN "Created tag: ${tag}"
    print_color $YELLOW "To push the tag, run: git push origin ${tag}"
    print_color $YELLOW "To trigger the release workflow, push to main branch with this tag"
}

# Function to show help
show_help() {
    echo "Flux Language Version Management Script"
    echo
    echo "Usage: $0 [command] [options]"
    echo
    echo "Commands:"
    echo "  current                    Show current version"
    echo "  suggest                    Suggest next version based on commits"
    echo "  create <version>           Create a new version tag"
    echo "  list                       List all version tags"
    echo "  help                       Show this help message"
    echo
    echo "Examples:"
    echo "  $0 current                 # Show current version"
    echo "  $0 suggest                 # Suggest next version"
    echo "  $0 create 1.2.3           # Create version 1.2.3"
    echo "  $0 list                    # List all versions"
    echo
    echo "Conventional Commit Patterns:"
    echo "  feat: new feature          → minor version bump"
    echo "  fix: bug fix              → patch version bump"
    echo "  perf: performance         → patch version bump"
    echo "  feat!: breaking change    → major version bump"
    echo "  BREAKING CHANGE: ...      → major version bump"
}

# Main script logic
case "${1:-help}" in
    "current")
        current_version=$(get_current_version)
        print_color $GREEN "Current version: ${current_version}"
        ;;
    "suggest")
        current_version=$(get_current_version)
        print_color $GREEN "Current version: ${current_version}"
        echo
        suggest_version "$current_version"
        ;;
    "create")
        if [ -z "$2" ]; then
            print_color $RED "Error: Version number required"
            echo "Usage: $0 create <version>"
            exit 1
        fi
        create_version "$2"
        ;;
    "list")
        print_color $GREEN "All version tags:"
        git tag -l "v*" --sort=-version:refname | head -20
        ;;
    "help"|*)
        show_help
        ;;
esac 