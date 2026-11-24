#!/usr/bin/env bash
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

# build-pages.sh OUT_DIR PKG_DIR
#
# Builds the Perci82 package repository site in OUT_DIR (replaced) from
# PKG_DIR, which holds one folder per release tag (system-vX.Y.Z,
# dev-vX.Y.Z) with that release's .deb, .rpm and checksums.txt. Folders
# without packages are skipped. The site holds:
#
#   apt/                 apt repository (suite "stable", component "main"),
#                        Release signed as InRelease and Release.gpg
#   rpm/                 dnf repository: signed RPMs and repodata/, with
#                        repomd.xml signed as repomd.xml.asc
#   latest/<app>.json    newest version of each app with packages
#   perci82.asc          the public signing key
#   perci82.sources      apt source file (deb822)
#   perci82.repo         dnf repository file
#   index.html           install instructions
#
# Environment:
#   GNUPGHOME             GnuPG home holding the secret signing key
#   REPO_PASSPHRASE_FILE  file whose first line is the key's passphrase
#   REPO_PUBLIC_KEY       public key to publish; the secret key must match
#                         it (default packaging/repo/perci82.asc)
#   SITE_URL              site address (default
#                         https://oito2.github.io/perci82)
set -Eeuo pipefail
shopt -s nullglob

readonly SITE_URL="${SITE_URL:-https://oito2.github.io/perci82}"
readonly PUBLIC_KEY="${REPO_PUBLIC_KEY:-packaging/repo/perci82.asc}"
readonly APPS=(system dev)
readonly APT_KEYRING="/usr/share/keyrings/perci82-archive-keyring.asc"
readonly RPM_KEY="/etc/pki/rpm-gpg/RPM-GPG-KEY-perci82"

die() {
    printf 'build-pages: %s\n' "$*" >&2
    exit 1
}

info() { printf 'build-pages: %s\n' "$*"; }

# Runs gpg with the signing key and its passphrase, non-interactively.
gpg_sign() {
    gpg --batch --yes --pinentry-mode loopback --passphrase-file "$REPO_PASSPHRASE_FILE" \
        --local-user "$fpr" "$@"
}

# Checks that every package in folder $1 is listed in its checksums.txt with
# a matching SHA-256.
verify_checksums() {
    local dir=$1 f name want got
    [[ -f "$dir/checksums.txt" ]] || die "$dir has no checksums.txt"
    for f in "$dir"/*.deb "$dir"/*.rpm; do
        name=$(basename -- "$f")
        want=$(awk -v n="$name" '$2 == n || $2 == "*" n { print $1; exit }' "$dir/checksums.txt")
        [[ -n "$want" ]] || die "$name is not listed in $dir/checksums.txt"
        got=$(sha256sum -- "$f" | awk '{ print $1 }')
        [[ "${got,,}" == "${want,,}" ]] || die "checksum mismatch for $name"
    done
}

main() {
    [[ $# -eq 2 ]] || die "usage: build-pages.sh OUT_DIR PKG_DIR"
    local out=$1 pkgs=$2
    : "${GNUPGHOME:?GNUPGHOME must hold the secret signing key}"
    : "${REPO_PASSPHRASE_FILE:?REPO_PASSPHRASE_FILE must name the passphrase file}"
    local cmd
    for cmd in gpg apt-ftparchive createrepo_c rpm rpmsign sha256sum gzip; do
        command -v "$cmd" >/dev/null || die "missing tool: $cmd"
    done
    [[ -d "$pkgs" ]] || die "$pkgs is not a folder"

    fpr=$(gpg --batch --show-keys --with-colons "$PUBLIC_KEY" | awk -F: '/^fpr:/ { print $10; exit }')
    [[ -n "$fpr" ]] || die "no key in $PUBLIC_KEY"
    gpg --batch --list-secret-keys "$fpr" >/dev/null 2>&1 || die "GNUPGHOME has no secret key $fpr"
    info "signing with $fpr"

    rm -rf -- "$out"
    mkdir -p -- "$out/apt/dists/stable/main/binary-amd64" "$out/rpm" "$out/latest"

    # Stage the packages of every release folder.
    local dir tag app f tags=()
    for dir in "$pkgs"/*/; do
        dir=${dir%/}
        tag=$(basename -- "$dir")
        [[ "$tag" =~ ^(system|dev)-v[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "unexpected folder $tag"
        app=${tag%%-v*}
        local debs=("$dir"/perci82-"$app"_*_amd64.deb) rpms=("$dir"/perci82-"$app"-[0-9]*.x86_64.rpm)
        if ((${#debs[@]} == 0 && ${#rpms[@]} == 0)); then
            info "$tag has no packages, skipped"
            continue
        fi
        ((${#debs[@]} > 0 && ${#rpms[@]} > 0)) || die "$tag must have both a .deb and a .rpm"
        verify_checksums "$dir"
        install -D -m 644 -t "$out/apt/pool/main/p/perci82-$app" -- "${debs[@]}"
        install -m 644 -t "$out/rpm" -- "${rpms[@]}"
        tags+=("$tag")
    done
    ((${#tags[@]} > 0)) || die "no packages found in $pkgs"

    # dnf: sign each package, check the signature, index and sign the index.
    info "signing $(find "$out/rpm" -name '*.rpm' | wc -l) RPMs"
    rpmsign --addsign \
        --define "_gpg_name $fpr" \
        --define "_gpg_path $GNUPGHOME" \
        --define "_gpg_sign_cmd_extra_args --pinentry-mode loopback --passphrase-file $REPO_PASSPHRASE_FILE" \
        "$out"/rpm/*.rpm >/dev/null
    local rpmdb
    rpmdb=$(mktemp -d)
    rpm --dbpath "$rpmdb" --import "$PUBLIC_KEY"
    for f in "$out"/rpm/*.rpm; do
        [[ "$(rpm --dbpath "$rpmdb" --checksig -- "$f")" == *"signatures OK"* ]] || die "bad RPM signature: $(basename -- "$f")"
    done
    rm -rf -- "$rpmdb"
    createrepo_c --quiet "$out/rpm"
    gpg_sign --armor --detach-sign --output "$out/rpm/repodata/repomd.xml.asc" "$out/rpm/repodata/repomd.xml"

    # apt: index the pool, write and sign the Release file.
    info "indexing the apt repository"
    (cd "$out/apt" && apt-ftparchive packages pool) >"$out/apt/dists/stable/main/binary-amd64/Packages"
    gzip -9 -n -k "$out/apt/dists/stable/main/binary-amd64/Packages"
    local release
    release=$(mktemp)
    apt-ftparchive \
        -o APT::FTPArchive::Release::Origin=Perci82 \
        -o APT::FTPArchive::Release::Label=Perci82 \
        -o APT::FTPArchive::Release::Suite=stable \
        -o APT::FTPArchive::Release::Codename=stable \
        -o APT::FTPArchive::Release::Architectures=amd64 \
        -o APT::FTPArchive::Release::Components=main \
        -o "APT::FTPArchive::Release::Description=Perci82 System and Perci82 Dev" \
        release "$out/apt/dists/stable" >"$release"
    install -m 644 -- "$release" "$out/apt/dists/stable/Release"
    rm -f -- "$release"
    gpg_sign --clearsign --output "$out/apt/dists/stable/InRelease" "$out/apt/dists/stable/Release"
    gpg_sign --armor --detach-sign --output "$out/apt/dists/stable/Release.gpg" "$out/apt/dists/stable/Release"

    # Check the signatures against the published key only.
    local keyring
    keyring=$(mktemp -d)
    gpg --batch --dearmor --output "$keyring/key.gpg" "$PUBLIC_KEY"
    gpgv --keyring "$keyring/key.gpg" "$out/apt/dists/stable/InRelease" 2>/dev/null || die "bad InRelease signature"
    gpgv --keyring "$keyring/key.gpg" "$out/apt/dists/stable/Release.gpg" "$out/apt/dists/stable/Release" 2>/dev/null || die "bad Release.gpg signature"
    gpgv --keyring "$keyring/key.gpg" "$out/rpm/repodata/repomd.xml.asc" "$out/rpm/repodata/repomd.xml" 2>/dev/null || die "bad repomd.xml signature"
    rm -rf -- "$keyring"

    # The newest version of each app.
    local latest
    for app in "${APPS[@]}"; do
        latest=$(printf '%s\n' "${tags[@]}" | grep "^$app-v" | sort -V | tail -n 1 || true)
        [[ -n "$latest" ]] || continue
        printf '{"app":"%s","version":"%s","tag":"%s"}\n' "$app" "v${latest#"$app"-v}" "$latest" >"$out/latest/$app.json"
    done

    # Client files and the instructions page.
    install -m 644 -- "$PUBLIC_KEY" "$out/perci82.asc"
    cat >"$out/perci82.sources" <<EOF
Types: deb
URIs: $SITE_URL/apt
Suites: stable
Components: main
Architectures: amd64
Signed-By: $APT_KEYRING
EOF
    cat >"$out/perci82.repo" <<EOF
[perci82]
name=Perci82
baseurl=$SITE_URL/rpm
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=file://$RPM_KEY
EOF
    write_index "$out/index.html"
    info "site written to $out (${tags[*]})"
}

# Writes the instructions page to $1.
write_index() {
    cat >"$1" <<EOF
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Perci82 packages</title>
<style>
  :root { color-scheme: light dark; --bg: #ffffff; --fg: #1f2328; --muted: #59636e; --code: #f6f8fa; }
  @media (prefers-color-scheme: dark) { :root { --bg: #0d1117; --fg: #e6edf3; --muted: #9198a1; --code: #161b22; } }
  body { margin: 0 auto; max-width: 48rem; padding: 2rem 1rem; background: var(--bg); color: var(--fg);
         font: 16px/1.6 system-ui, sans-serif; }
  h1, h2 { line-height: 1.25; }
  p.muted { color: var(--muted); }
  pre { background: var(--code); padding: 0.75rem 1rem; border-radius: 6px; overflow-x: auto; }
  code { font-family: ui-monospace, monospace; font-size: 0.9em; }
</style>
</head>
<body>
<h1>Perci82 packages</h1>
<p>Package repository of <a href="https://github.com/oito2/perci82">Perci82 System and Perci82 Dev</a>
for Debian 13, Ubuntu 24.04, Linux Mint 22 and Fedora 40 or newer (amd64).</p>

<h2>Install</h2>
<pre><code>curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/oito2/perci82/main/install.sh | bash</code></pre>
<p>The script asks which apps to install, adds this repository and installs them. Updates then come with
<code>apt upgrade</code> or <code>dnf upgrade</code>.</p>

<h2>Add the repository by hand</h2>
<p>apt (Debian, Ubuntu, Linux Mint, Zorin OS, Pop!_OS):</p>
<pre><code>sudo curl -fsSLo $APT_KEYRING $SITE_URL/perci82.asc
sudo curl -fsSLo /etc/apt/sources.list.d/perci82.sources $SITE_URL/perci82.sources
sudo apt update
sudo apt install perci82-system perci82-dev</code></pre>
<p>dnf (Fedora):</p>
<pre><code>sudo curl -fsSLo $RPM_KEY $SITE_URL/perci82.asc
sudo curl -fsSLo /etc/yum.repos.d/perci82.repo $SITE_URL/perci82.repo
sudo dnf install perci82-system perci82-dev</code></pre>

<h2>Signing key</h2>
<p>Packages and indexes are signed with the key <code>$fpr</code>.</p>
</body>
</html>
EOF
}

main "$@"
