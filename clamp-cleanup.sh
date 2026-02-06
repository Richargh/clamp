#!/bin/bash

usage() {
    echo "Usage: $(basename "$0") [-h|--help]"
    echo ""
    echo "Interactive cleanup tool for Docker volumes created by claude-clamp and open-clamp."
    echo ""
    echo "Options:"
    echo "  -h, --help    Show this help message"
    echo ""
    echo "Controls:"
    echo "  ↑/↓           Navigate the list"
    echo "  Space         Toggle selection"
    echo "  A             Select all"
    echo "  N             Select none"
    echo "  Enter         Confirm and delete selected"
    echo "  Q             Quit without deleting"
}

case "${1:-}" in
    -h|--help)
        usage
        exit 0
        ;;
    -*)
        echo "Unknown option: $1"
        usage
        exit 1
        ;;
esac

# ANSI escape codes
ESC=$'\e'
CURSOR_UP="${ESC}[A"
CLEAR_LINE="${ESC}[2K"
HIDE_CURSOR="${ESC}[?25l"
SHOW_CURSOR="${ESC}[?25h"
BOLD="${ESC}[1m"
RESET="${ESC}[0m"
GREEN="${ESC}[32m"
RED="${ESC}[31m"
CYAN="${ESC}[36m"

# Global arrays for state
ITEMS=()
SELECTED=()
CURSOR=0

# Restore cursor on exit
cleanup() {
    printf "%s" "$SHOW_CURSOR"
    stty echo 2>/dev/null || true
}
trap cleanup EXIT

# Get all docker volumes
get_volumes() {
    docker volume ls --format '{{.Name}}' 2>/dev/null
}

# Extract project names from volume names
get_projects() {
    local volumes
    volumes=$(get_volumes)

    local projects=()

    while IFS= read -r vol; do
        local project=""
        case "$vol" in
            *-node-modules)
                project="${vol%-node-modules}"
                ;;
            *-gradle-build)
                project="${vol%-gradle-build}"
                ;;
            *-gradle-cache)
                project="${vol%-gradle-cache}"
                ;;
        esac

        if [[ -n "$project" ]]; then
            local found=false
            for p in "${projects[@]}"; do
                [[ "$p" == "$project" ]] && found=true && break
            done
            $found || projects+=("$project")
        fi
    done <<< "$volumes"

    printf '%s\n' "${projects[@]}" | sort
}

# Check if global claude-clamp volume exists
has_claude_global_volume() {
    docker volume ls --format '{{.Name}}' 2>/dev/null | grep -qx "claude-clamp"
}

# Check if global opencode-clamp volume exists
has_opencode_global_volume() {
    docker volume ls --format '{{.Name}}' 2>/dev/null | grep -qx "opencode-clamp"
}

# Get volumes for a project
get_project_volumes() {
    local project="$1"
    local volumes=()

    for suffix in node-modules gradle-build gradle-cache; do
        local vol="${project}-${suffix}"
        if docker volume ls --format '{{.Name}}' | grep -qx "$vol"; then
            volumes+=("$vol")
        fi
    done

    # Check for Claude per-project auth volume
    local claude_auth_vol="claude-clamp-${project}"
    if docker volume ls --format '{{.Name}}' | grep -qx "$claude_auth_vol"; then
        volumes+=("$claude_auth_vol")
    fi

    # Check for OpenCode per-project auth volume
    local opencode_auth_vol="opencode-clamp-${project}"
    if docker volume ls --format '{{.Name}}' | grep -qx "$opencode_auth_vol"; then
        volumes+=("$opencode_auth_vol")
    fi

    printf '%s\n' "${volumes[@]}"
}

# Read single keypress
read_key() {
    local key
    IFS= read -rsn1 key

    if [[ "$key" == "$ESC" ]]; then
        local seq1 seq2
        read -rsn1 -t 1 seq1 || true
        read -rsn1 -t 1 seq2 || true
        if [[ "$seq1" == "[" ]]; then
            case "$seq2" in
                'A') echo "UP"; return ;;
                'B') echo "DOWN"; return ;;
            esac
        fi
        echo "ESC"
    elif [[ "$key" == "" ]]; then
        echo "ENTER"
    elif [[ "$key" == " " ]]; then
        echo "SPACE"
    elif [[ "$key" == "a" || "$key" == "A" ]]; then
        echo "ALL"
    elif [[ "$key" == "n" || "$key" == "N" ]]; then
        echo "NONE"
    elif [[ "$key" == "q" || "$key" == "Q" ]]; then
        echo "QUIT"
    else
        echo "OTHER"
    fi
}

# Draw the checkbox list
draw_list() {
    local i
    for ((i = 0; i < ${#ITEMS[@]}; i++)); do
        printf "%s" "$CLEAR_LINE"

        local checkbox="[ ]"
        [[ "${SELECTED[$i]}" == "1" ]] && checkbox="${GREEN}[x]${RESET}"

        local prefix="   "
        [[ $i -eq $CURSOR ]] && prefix="${CYAN}>${RESET}  "

        printf "%s%s %s\n" "$prefix" "$checkbox" "${ITEMS[$i]}"
    done
}

# Main interactive selection - returns 0 on confirm, 1 on cancel
interactive_select() {
    local num_items=${#ITEMS[@]}

    printf "%s" "$HIDE_CURSOR"
    stty -echo 2>/dev/null || true

    echo ""
    echo "${BOLD}Select projects to clean up:${RESET}"
    echo "  ↑/↓: Navigate  Space: Toggle  A: All  N: None  Enter: Confirm  Q: Quit"
    echo ""

    draw_list

    while true; do
        for ((i = 0; i < num_items; i++)); do
            printf "%s" "$CURSOR_UP"
        done

        draw_list

        local key
        key=$(read_key)

        case "$key" in
            UP)
                [[ $CURSOR -gt 0 ]] && CURSOR=$((CURSOR - 1))
                ;;
            DOWN)
                [[ $CURSOR -lt $((num_items - 1)) ]] && CURSOR=$((CURSOR + 1))
                ;;
            SPACE)
                if [[ "${SELECTED[$CURSOR]}" == "1" ]]; then
                    SELECTED[CURSOR]="0"
                else
                    SELECTED[CURSOR]="1"
                fi
                ;;
            ALL)
                for ((i = 0; i < num_items; i++)); do
                    SELECTED[i]="1"
                done
                ;;
            NONE)
                for ((i = 0; i < num_items; i++)); do
                    SELECTED[i]="0"
                done
                ;;
            ENTER)
                echo ""
                printf "%s" "$SHOW_CURSOR"
                stty echo 2>/dev/null || true
                return 0
                ;;
            QUIT|ESC)
                echo ""
                printf "%s" "$SHOW_CURSOR"
                stty echo 2>/dev/null || true
                return 1
                ;;
        esac
    done
}

# Main
main() {
    echo "${BOLD}Clamp Volume Cleanup${RESET}"
    echo ""

    local projects_list
    projects_list=$(get_projects)

    if [[ -z "$projects_list" ]]; then
        local has_any_global=false
        has_claude_global_volume && has_any_global=true
        has_opencode_global_volume && has_any_global=true

        if ! $has_any_global; then
            echo "No clamp project volumes found."
            exit 0
        fi
    fi

    while IFS= read -r project; do
        [[ -n "$project" ]] && ITEMS+=("$project")
    done <<< "$projects_list"

    # Track global volume positions
    local claude_global_idx=-1
    local opencode_global_idx=-1

    if has_claude_global_volume; then
        claude_global_idx=${#ITEMS[@]}
        ITEMS+=("[GLOBAL] claude-clamp config")
    fi

    if has_opencode_global_volume; then
        opencode_global_idx=${#ITEMS[@]}
        ITEMS+=("[GLOBAL] opencode-clamp config")
    fi

    if [[ ${#ITEMS[@]} -eq 0 ]]; then
        echo "No clamp volumes found."
        exit 0
    fi

    for ((i = 0; i < ${#ITEMS[@]}; i++)); do
        SELECTED+=("0")
    done

    if ! interactive_select; then
        echo "Cancelled."
        exit 0
    fi

    local volumes_to_delete=()

    for ((i = 0; i < ${#ITEMS[@]}; i++)); do
        if [[ "${SELECTED[$i]}" == "1" ]]; then
            if [[ $i -eq $claude_global_idx ]]; then
                volumes_to_delete+=("claude-clamp")
            elif [[ $i -eq $opencode_global_idx ]]; then
                volumes_to_delete+=("opencode-clamp")
            else
                local project="${ITEMS[$i]}"
                while IFS= read -r vol; do
                    [[ -n "$vol" ]] && volumes_to_delete+=("$vol")
                done <<< "$(get_project_volumes "$project")"
            fi
        fi
    done

    if [[ ${#volumes_to_delete[@]} -eq 0 ]]; then
        echo "No volumes selected."
        exit 0
    fi

    echo ""
    echo "${BOLD}The following volumes will be deleted:${RESET}"
    for vol in "${volumes_to_delete[@]}"; do
        echo "  - $vol"
    done
    echo ""

    read -rp "Proceed? [y/N] " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        echo "Cancelled."
        exit 0
    fi

    echo ""
    for vol in "${volumes_to_delete[@]}"; do
        printf "Deleting %s... " "$vol"
        if docker volume rm "$vol" 2>/dev/null; then
            echo "${GREEN}done${RESET}"
        else
            echo "${RED}failed${RESET} (may be in use)"
        fi
    done

    echo ""
    echo "${GREEN}Cleanup complete.${RESET}"
}

main "$@"
