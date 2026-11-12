// Copyright (C) 2026  oito2
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package rust installs, updates and removes the Rust toolchain through
// rustup, for the current user: rustup and cargo in ~/.cargo, toolchains
// in ~/.rustup. The installer (rustup-init) is a pinned release checked
// against a pinned SHA-256 before it runs.
package rust

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/oito2/perci82/internal/dev/nettimeout"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/fsutil"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/shellrc"
	"github.com/oito2/perci82/internal/ui"
)

// rustupVersion is the rustup-init release that is downloaded.
const rustupVersion = "1.29.1"

// rustupChecksums maps a rustup target to the SHA-256 of its rustup-init
// for rustupVersion, as published next to it on static.rust-lang.org.
var rustupChecksums = map[string]string{
	"x86_64-unknown-linux-gnu":  "dda7234360b7f578ca8b0ddcb80145646fa61a67c1720a5abc7051b35c9fcb71",
	"aarch64-unknown-linux-gnu": "15f6e4ce9f583b929c996c91562bad6d4454f3281de858b02cdfdef615fac433",
}

// downloadBase is the root rustup-init is downloaded from; a variable so
// tests can point it at a local server.
var downloadBase = "https://static.rust-lang.org/rustup/archive"

// Components are installed on top of rustup's default profile (rustc,
// cargo, rust-std, rust-docs, rustfmt, clippy).
var components = []string{"rust-analyzer"}

const (
	// rcComment and pathEntry are the lines written to the shell rc file;
	// the quoted form also works in fish.
	rcComment = "# Rust"
	pathEntry = `export PATH="$HOME/.cargo/bin:$PATH"`
)

// target returns the rustup target of this machine.
func target() string {
	if runtime.GOARCH == "arm64" {
		return "aarch64-unknown-linux-gnu"
	}
	return "x86_64-unknown-linux-gnu"
}

// cargoBin returns the folder rustup puts its commands in: $CARGO_HOME/bin,
// or ~/.cargo/bin.
func cargoBin() (string, error) {
	if dir := os.Getenv("CARGO_HOME"); dir != "" {
		return filepath.Join(dir, "bin"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home folder: %w", err)
	}
	return filepath.Join(home, ".cargo", "bin"), nil
}

// IsInstalled reports whether rustup and rustc are in the cargo bin folder.
func IsInstalled() bool {
	bin, err := cargoBin()
	if err != nil {
		return false
	}
	for _, name := range []string{"rustup", "rustc"} {
		if _, err := os.Stat(filepath.Join(bin, name)); err != nil {
			return false
		}
	}
	return true
}

// Install downloads rustup-init, verifies it and runs it without changing
// the shell files (the stable toolchain with the default profile plus
// components), then adds the cargo bin folder to PATH in the shell rc file.
func Install(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	t := target()
	checksum, ok := rustupChecksums[t]
	if !ok {
		return fmt.Errorf("no Rust installer for this architecture: %s", t)
	}
	tmp, err := os.CreateTemp("", "rustup-init-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	url := downloadBase + "/" + rustupVersion + "/" + t + "/rustup-init"
	ui.Info(stdout, i18n.Tr("sdk.run.rustDownload", i18n.Args{"version": rustupVersion}))
	if err := exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, "curl", "-fsSL", "--connect-timeout", "10", "--max-time", "600", "-o", tmpPath, "--", url); err != nil {
		return fmt.Errorf("download rustup-init: %w", err)
	}
	if !exe.DryRun {
		ui.Info(stdout, i18n.Tr("sdk.run.rustVerify", nil))
		if err := fsutil.VerifyFile(tmpPath, sha256.New(), checksum); err != nil {
			return fmt.Errorf("rustup-init integrity check failed: %w", err)
		}
		if err := os.Chmod(tmpPath, 0o700); err != nil {
			return err
		}
	}

	args := []string{"-y", "--no-modify-path", "--profile", "default", "--default-toolchain", "stable"}
	for _, c := range components {
		args = append(args, "--component", c)
	}
	ui.Info(stdout, i18n.Tr("sdk.run.rustInstall", nil))
	runCtx, cancel := nettimeout.Context(ctx)
	defer cancel()
	if err := exe.Run(runCtx, executor.Options{Stdout: stdout, Stderr: stdout}, tmpPath, args...); err != nil {
		return fmt.Errorf("install Rust: %w", err)
	}
	ensurePath(stdout)
	ui.Success(stdout, i18n.Tr("sdk.run.rustInstalled", nil))
	return nil
}

// Update updates rustup and the installed toolchains ("rustup update"),
// and adds any component missing from an older install.
func Update(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	rustup, err := rustupPath()
	if err != nil {
		return err
	}
	runCtx, cancel := nettimeout.Context(ctx)
	defer cancel()
	ui.Info(stdout, i18n.Tr("sdk.run.rustUpdate", nil))
	if err := exe.Run(runCtx, executor.Options{Stdout: stdout, Stderr: stdout}, rustup, "update"); err != nil {
		return fmt.Errorf("update Rust: %w", err)
	}
	if err := exe.Run(runCtx, executor.Options{Stdout: stdout, Stderr: stdout}, rustup, append([]string{"component", "add"}, components...)...); err != nil {
		return fmt.Errorf("add Rust components: %w", err)
	}
	ensurePath(stdout)
	return nil
}

// Uninstall removes rustup, every toolchain and ~/.cargo ("rustup self
// uninstall"), then the PATH line written by Install.
func Uninstall(ctx context.Context, exe *executor.Executor, stdout io.Writer) error {
	rustup, err := rustupPath()
	if err != nil {
		return err
	}
	ui.Info(stdout, i18n.Tr("sdk.run.rustRemoving", nil))
	if err := exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, rustup, "self", "uninstall", "-y"); err != nil {
		return fmt.Errorf("remove Rust: %w", err)
	}
	removePath(stdout)
	ui.Success(stdout, i18n.Tr("sdk.run.rustRemoved", nil))
	return nil
}

// rustupPath returns the rustup command in the cargo bin folder.
func rustupPath() (string, error) {
	bin, err := cargoBin()
	if err != nil {
		return "", err
	}
	return filepath.Join(bin, "rustup"), nil
}

// ensurePath adds the cargo bin folder to PATH in the current shell's rc
// file.
func ensurePath(stdout io.Writer) {
	home, err := os.UserHomeDir()
	if err != nil {
		ui.Warning(stdout, i18n.Tr("dev.noHome", nil))
		return
	}
	rc := shellrc.File(home)
	appended, err := shellrc.AppendIfMissing(rc, rcComment, pathEntry)
	if err != nil {
		ui.Warning(stdout, i18n.Tr("dev.rcFailed", i18n.Args{"path": rc, "error": err.Error()}))
		return
	}
	if appended {
		ui.Info(stdout, i18n.Tr("dev.pathUpdated", i18n.Args{"path": rc}))
	}
}

// removePath removes the lines ensurePath writes from ~/.bashrc and the
// current shell's rc file.
func removePath(stdout io.Writer) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	candidates := shellrc.Dedup([]string{filepath.Join(home, ".bashrc"), shellrc.File(home)})
	for _, res := range shellrc.RemoveEntry(candidates, rcComment, pathEntry) {
		if res.Err != nil {
			ui.Warning(stdout, i18n.Tr("dev.rcFailed", i18n.Args{"path": res.Path, "error": res.Err.Error()}))
			continue
		}
		ui.Info(stdout, i18n.Tr("dev.pathRemoved", i18n.Args{"path": res.Path}))
	}
}
