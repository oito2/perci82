#!/usr/bin/env bash
#
# Copyright (C) 2026  oito2
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

# =============================================================================
# Description : Installs Perci82 System and/or Perci82 Dev from the Perci82
#               package repository: adds the repository (apt or dnf) and its
#               signing key, installs the chosen packages and offers to
#               remove the old Perci (prci), keeping its settings in ~/.perci.
# Usage       : install.sh [system] [dev] [all]
#               Without arguments it asks which apps to install.
# =============================================================================
set -Eeuo pipefail
shopt -s inherit_errexit

readonly SITE_URL="https://oito2.github.io/perci82"
readonly KEY_URL="$SITE_URL/perci82.asc"
readonly KEY_FINGERPRINT="A1B1AC75B5350344B6C8043CFF9E1B0F3C2A545F"
readonly APT_KEYRING="/usr/share/keyrings/perci82-archive-keyring.asc"
readonly APT_SOURCES="/etc/apt/sources.list.d/perci82.sources"
readonly RPM_KEY="/etc/pki/rpm-gpg/RPM-GPG-KEY-perci82"
readonly DNF_REPO="/etc/yum.repos.d/perci82.repo"
readonly CURL=(curl --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 10 --max-time 600 -fsSL)

# The old Perci (oito2/perci), offered for removal.
readonly INSTALL_DIR="/usr/local/bin"
readonly HICOLOR_DIR="/usr/share/icons/hicolor"
readonly LEGACY_BINARY="prci"
readonly LEGACY_ALIAS="perci"
readonly LEGACY_DESKTOP="usr/share/applications/perci.desktop"
readonly LEGACY_PIXMAP="usr/share/pixmaps/perci.png"
readonly LEGACY_ICON_SIZES=(512 256 128 64 48 32)
readonly LEGACY_AUTOSTART=".config/autostart/perci.desktop"

# --- Output helpers (data is printed with %s, never interpreted) ---
info() { printf '\033[1;34m[INFO]\033[0m %s\n' "$*"; }
success() { printf '\033[1;32m[SUCCESS]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[WARNING]\033[0m %s\n' "$*" >&2; }
err() { printf '\033[1;31m[ERROR]\033[0m %s\n' "$*" >&2; }
die() {
    err "$*"
    exit 1
}

usage() {
    cat <<'EOF'
Usage: install.sh [system] [dev] [all]

Installs Perci82 System (system), Perci82 Dev (dev) or both (all) from the
Perci82 package repository. Without arguments it asks which apps to install.

  curl -fsSL https://raw.githubusercontent.com/oito2/perci82/main/install.sh | bash
  curl -fsSL https://raw.githubusercontent.com/oito2/perci82/main/install.sh | bash -s -- dev
EOF
}

# Runs "$@" as root: directly when already root, through sudo otherwise.
as_root() {
    if [[ $EUID -eq 0 ]]; then
        "$@"
    else
        sudo "$@"
    fi
}

# Prints the package names for the app arguments "$@" (system, dev, all),
# one per line, in a fixed order and without repeats. Fails on an unknown
# argument.
packages_for() {
    local arg system=0 dev=0
    for arg in "$@"; do
        case "${arg,,}" in
            system) system=1 ;;
            dev) dev=1 ;;
            all) system=1 dev=1 ;;
            *)
                err "Unknown app '$arg': use system, dev or all."
                return 1
                ;;
        esac
    done
    if ((system)); then echo perci82-system; fi
    if ((dev)); then echo perci82-dev; fi
}

# Asks on the terminal which apps to install (the script itself may be read
# from a pipe) and prints their package names. Returns 2 when there is no
# terminal to ask on and 1 after three invalid answers.
ask_packages() {
    local answer=""
    if ! { exec 3</dev/tty; } 2>/dev/null; then
        return 2
    fi
    {
        printf 'Which Perci82 apps do you want to install?\n'
        printf '  1) Perci82 System — Linux administration\n'
        printf '  2) Perci82 Dev    — development environment\n'
        printf '  3) Both\n'
    } >/dev/tty
    for _ in 1 2 3; do
        printf 'Choose 1, 2 or 3: ' >/dev/tty
        read -r answer <&3 || answer=""
        case "$answer" in
            1) packages_for system ;;
            2) packages_for dev ;;
            3) packages_for all ;;
            *) continue ;;
        esac
        exec 3<&-
        return 0
    done
    exec 3<&-
    return 1
}

# Prints "apt" or "dnf", the package manager this system uses, or nothing.
package_manager() {
    if command -v apt-get &>/dev/null; then
        echo apt
    elif command -v dnf &>/dev/null; then
        echo dnf
    fi
}

# Succeeds when the key file $1 holds exactly one key, with fingerprint
# KEY_FINGERPRINT. Uses a throwaway GnuPG home, never the user's keyring.
verify_key() {
    local home fprs
    home=$(mktemp -d)
    fprs=$(GNUPGHOME="$home" gpg --batch --quiet --show-keys --with-colons "$1" 2>/dev/null |
        awk -F: '$1 == "pub" { want = 1; next } want && $1 == "fpr" { print $10; want = 0 }') || true
    rm -rf -- "$home"
    [[ "$fprs" == "$KEY_FINGERPRINT" ]]
}

# Prints the apt source file of the repository (deb822).
apt_sources_content() {
    cat <<EOF
Types: deb
URIs: $SITE_URL/apt
Suites: stable
Components: main
Architectures: amd64
Signed-By: $APT_KEYRING
EOF
}

# Prints the dnf repository file.
dnf_repo_content() {
    cat <<EOF
[perci82]
name=Perci82
baseurl=$SITE_URL/rpm
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=file://$RPM_KEY
EOF
}

# Installs the verified key $1 and the repository file of package manager
# $2, both owned by root with mode 0644.
add_repository() {
    local key=$1 pm=$2 repo_file
    repo_file="$_tmpdir/repo"
    if [[ "$pm" == apt ]]; then
        apt_sources_content >"$repo_file"
        as_root install -D -m 0644 -o root -g root -T -- "$key" "$APT_KEYRING"
        as_root install -D -m 0644 -o root -g root -T -- "$repo_file" "$APT_SOURCES"
    else
        dnf_repo_content >"$repo_file"
        as_root install -D -m 0644 -o root -g root -T -- "$key" "$RPM_KEY"
        as_root rpm --import "$RPM_KEY"
        as_root install -D -m 0644 -o root -g root -T -- "$repo_file" "$DNF_REPO"
    fi
}

# Reports the packages in "$@" that are already installed: the install then
# updates them when a newer version exists.
report_installed() {
    local pm=$1 pkg
    shift
    for pkg in "$@"; do
        if [[ "$pm" == apt ]]; then
            dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q 'install ok installed' || continue
        else
            rpm -q "$pkg" &>/dev/null || continue
        fi
        info "$pkg is already installed — it is updated if a newer version exists."
    done
}

# Prints, one per line, the files of the old Perci found under the root
# folder $1 ("" for the real system): its binary, the alias when it is a
# symlink, its menu entry and its icons.
legacy_system_paths() {
    local root="$1" size path
    path="$root$INSTALL_DIR/$LEGACY_BINARY"
    if [[ -e "$path" || -L "$path" ]]; then printf '%s\n' "$path"; fi
    path="$root$INSTALL_DIR/$LEGACY_ALIAS"
    if [[ -L "$path" ]]; then printf '%s\n' "$path"; fi
    for path in "$root/$LEGACY_DESKTOP" "$root/$LEGACY_PIXMAP"; do
        if [[ -e "$path" ]]; then printf '%s\n' "$path"; fi
    done
    for size in "${LEGACY_ICON_SIZES[@]}"; do
        path="$root$HICOLOR_DIR/${size}x${size}/apps/perci.png"
        if [[ -e "$path" ]]; then printf '%s\n' "$path"; fi
    done
}

# Prints the old Perci's tray autostart entry under the home folder $1,
# when it exists.
legacy_user_paths() {
    local path="$1/$LEGACY_AUTOSTART"
    if [[ -e "$path" ]]; then printf '%s\n' "$path"; fi
}

# Asks the yes/no question $1 on the terminal (the script itself may be
# read from a pipe). Succeeds on yes; returns 1 on no and 2 when there is
# no terminal to ask on.
ask_yes_no() {
    local answer=""
    if ! { exec 3</dev/tty; } 2>/dev/null; then
        return 2
    fi
    printf '%s [y/N] ' "$1" >/dev/tty
    read -r answer <&3 || answer=""
    exec 3<&-
    [[ "${answer,,}" =~ ^(y|yes|s|sim)$ ]]
}

# Offers to remove the old Perci when it is installed under the root
# folder $1 ("" for the real system) and the home folder $2. Its settings
# in ~/.perci stay: both Perci82 applications read them.
remove_legacy_perci() {
    local system_paths=() user_paths=() rc=0
    mapfile -t system_paths < <(legacy_system_paths "$1")
    mapfile -t user_paths < <(legacy_user_paths "$2")
    if [[ ${#system_paths[@]} -eq 0 && ${#user_paths[@]} -eq 0 ]]; then
        return 0
    fi

    info "The old Perci is installed:"
    printf '    %s\n' "${system_paths[@]}" "${user_paths[@]}"
    ask_yes_no "Remove it? Its settings in ~/.perci are kept." || rc=$?
    case "$rc" in
        0) ;;
        2)
            warn "No terminal to ask on — the old Perci was kept. Remove the files above to uninstall it."
            return 0
            ;;
        *)
            info "The old Perci was kept."
            return 0
            ;;
    esac

    if [[ ${#system_paths[@]} -gt 0 ]]; then
        as_root rm -f -- "${system_paths[@]}" || {
            err "Failed to remove the old Perci."
            return 0
        }
        if command -v gtk-update-icon-cache >/dev/null 2>&1; then
            as_root gtk-update-icon-cache -q -t -f "$HICOLOR_DIR" || true
        fi
    fi
    if [[ ${#user_paths[@]} -gt 0 ]]; then
        rm -f -- "${user_paths[@]}" || err "Failed to remove the old Perci's autostart entry."
    fi
    success "The old Perci was removed (settings kept in ~/.perci)."
}

main() {
    local pkgs=() pm chosen rc=0 arg
    for arg in "$@"; do
        if [[ "$arg" == -h || "$arg" == --help ]]; then
            usage
            return 0
        fi
    done

    if [[ $# -gt 0 ]]; then
        chosen=$(packages_for "$@") || {
            usage >&2
            exit 1
        }
    else
        chosen=$(ask_packages) || rc=$?
        case "$rc" in
            0) ;;
            2)
                usage >&2
                die "No terminal to ask on: pass the apps as arguments (system, dev or all)."
                ;;
            *) die "No valid choice: run the script again and choose 1, 2 or 3." ;;
        esac
    fi
    mapfile -t pkgs <<<"$chosen"

    [[ "$(uname -m)" == x86_64 ]] || die "Perci82 only publishes amd64 (x86_64) packages; this machine is $(uname -m)."
    pm=$(package_manager)
    [[ -n "$pm" ]] || die "Perci82 needs apt (Debian, Ubuntu, Linux Mint, Zorin OS, Pop!_OS) or dnf (Fedora)."
    local cmd
    for cmd in curl gpg; do
        command -v "$cmd" &>/dev/null || die "Required tool '$cmd' is not installed. Please install it first."
    done
    if [[ "$pm" == apt ]] && ! LC_ALL=C apt-cache policy libwebkitgtk-6.0-4 2>/dev/null | grep -q 'Candidate: [0-9]'; then
        die "This release has no WebKitGTK 6.0 (libwebkitgtk-6.0-4): Perci82 needs Debian 13, Ubuntu 24.04, Linux Mint 22 or newer."
    fi

    _tmpdir=$(mktemp -d -t perci82-install-XXXXXX)
    info "Downloading the Perci82 repository key..."
    "${CURL[@]}" -o "$_tmpdir/perci82.asc" "$KEY_URL"
    verify_key "$_tmpdir/perci82.asc" || die "The repository key does not have the expected fingerprint $KEY_FINGERPRINT. Aborting."
    success "Repository key verified ($KEY_FINGERPRINT)."

    info "Adding the Perci82 $pm repository (administrator password needed)..."
    add_repository "$_tmpdir/perci82.asc" "$pm"
    report_installed "$pm" "${pkgs[@]}"

    info "Installing ${pkgs[*]}..."
    if [[ "$pm" == apt ]]; then
        as_root apt-get update
        as_root apt-get install -y "${pkgs[@]}"
    else
        as_root dnf install -y --refresh "${pkgs[@]}"
    fi

    remove_legacy_perci "" "$HOME"

    success "Installed: ${pkgs[*]}."
    if [[ "$pm" == apt ]]; then
        info "Open the apps from the application menu. Updates come with: sudo apt update && sudo apt upgrade"
    else
        info "Open the apps from the application menu. Updates come with: sudo dnf upgrade"
    fi
}

# Runs only when executed (also from a pipe), not when sourced. The error
# trap reports only in the main shell, so a failure inside a command
# substitution is reported once.
if [[ "${BASH_SOURCE[0]:-$0}" == "$0" ]]; then
    trap '((BASH_SUBSHELL == 0)) && printf "\n\033[0;31mError at line %d\033[0m\n" "$LINENO" >&2' ERR
    trap '[[ -n "${_tmpdir:-}" ]] && rm -rf -- "$_tmpdir"' EXIT
    main "$@"
fi
