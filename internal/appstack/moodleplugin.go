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

package appstack

// Development folder mounts: a development folder is bind-mounted on the
// host over an empty folder inside an app's project root — for a Moodle
// app, a plugin folder (for example html/<app>/local/<name>); for the other
// app types, a folder the user names. The mount is a systemd mount unit ordered
// before docker.service, so it is back after a reboot before any container
// starts. Containers see host mounts that already exist under their own
// bind mounts when they start, so a running app container (and Nginx)
// picks a new mount up on its next restart. Inside the container the
// mounted folder is a real folder, so PHP's __DIR__ stays inside the
// project tree.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// systemdUnitDir is where the mount units are installed.
var systemdUnitDir = "/etc/systemd/system"

// Status values of a Moodle plugin mount, see MoodlePluginStatuses.
const (
	PluginApplied  = "applied"  // the app container (and Nginx, when running) see the plugin
	PluginPending  = "pending"  // mounted on the host, waiting for a container restart
	PluginStopped  = "stopped"  // mounted on the host, the app container is not running
	PluginInactive = "inactive" // the host mount is not active
	PluginNoApp    = "missing"  // the app container does not exist
)

// pluginNamePattern is Moodle's rule for a plugin name (the part after
// the type in its component).
var pluginNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// componentPattern finds $plugin->component in a version.php.
var componentPattern = regexp.MustCompile(`\$plugin->component\s*=\s*['"]([a-z][a-z0-9]*)_([a-z][a-z0-9_]*)['"]`)

// MoodleApps returns the registered Moodle containers.
func MoodleApps(apps []config.AppContainer) []config.AppContainer {
	var out []config.AppContainer
	for _, a := range apps {
		if a.Type == config.AppTypeMoodle {
			out = append(out, a)
		}
	}
	return out
}

// MoodlePluginTypes reads the plugin types a Moodle accepts from its
// lib/components.json: type → folder relative to the app's html folder
// (for example "local" → "local" on Moodle 4.x, "public/local" on 5.1+).
func MoodlePluginTypes(workspace, folder string) (map[string]string, error) {
	path := filepath.Join(AppHTMLDir(workspace, folder), "lib", "components.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Tr("devfolder.noPluginTypes", i18n.Args{"app": folder, "path": path}), err)
	}
	var parsed struct {
		PluginTypes map[string]string `json:"plugintypes"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("invalid components.json in %s: %w", path, err)
	}
	if len(parsed.PluginTypes) == 0 {
		return nil, fmt.Errorf("no plugin types in %s", path)
	}
	return parsed.PluginTypes, nil
}

// DevPluginComponent returns the type and name declared by
// $plugin->component in dev/version.php, or empty strings when the file
// does not exist or declares no component.
func DevPluginComponent(dev string) (ptype, name string) {
	data, err := os.ReadFile(filepath.Join(dev, "version.php"))
	if err != nil {
		return "", ""
	}
	m := componentPattern.FindSubmatch(data)
	if m == nil {
		return "", ""
	}
	return string(m[1]), string(m[2])
}

// MoodlePluginPlan is a validated mount: Dest is the folder inside the
// app's html folder (RelPath relative to it) where DevPath will appear.
// Component is the Moodle plugin's component, empty for a mount into
// another app type.
type MoodlePluginPlan struct {
	App       string
	Component string
	DevPath   string
	Dest      string
	RelPath   string
	HTML      string // the app's html folder, links resolved: Dest must stay inside it
}

// PlanMoodlePlugin validates mounting dev as the plugin ptype_name into
// the Moodle app folder. dev must be an existing folder outside the app's
// code; the destination must not exist yet or be an empty folder, and no
// other record may use it. When dev/version.php declares a component, it
// must be ptype_name.
func PlanMoodlePlugin(cfg *config.Config, dev, folder, ptype, name string) (MoodlePluginPlan, error) {
	workspace, err := resolveWorkspace(cfg)
	if err != nil {
		return MoodlePluginPlan{}, err
	}
	app, found := FindAppByFolder(cfg.Docker.Apps, folder)
	if !found || app.Type != config.AppTypeMoodle {
		return MoodlePluginPlan{}, errors.New(i18n.Tr("devfolder.notMoodle", i18n.Args{"app": folder}))
	}
	if !pluginNamePattern.MatchString(name) {
		return MoodlePluginPlan{}, i18n.NewError("devfolder.badPluginName")
	}
	types, err := MoodlePluginTypes(workspace, folder)
	if err != nil {
		return MoodlePluginPlan{}, err
	}
	typeDir, ok := types[ptype]
	if !ok {
		return MoodlePluginPlan{}, errors.New(i18n.Tr("devfolder.noPluginType", i18n.Args{"type": ptype, "app": folder}))
	}
	if declType, declName := DevPluginComponent(dev); declType != "" && (declType != ptype || declName != name) {
		return MoodlePluginPlan{}, errors.New(i18n.Tr("devfolder.declares", i18n.Args{"component": declType + "_" + declName}))
	}

	relPath := filepath.Join(typeDir, name)
	if typeDir == "" || filepath.IsAbs(typeDir) || !filepath.IsLocal(relPath) {
		return MoodlePluginPlan{}, errors.New(i18n.Tr("devfolder.typeOutside", i18n.Args{"type": ptype}))
	}
	return planMount(cfg, workspace, dev, folder, relPath, ptype+"_"+name)
}

// DevFolderTarget is where a development folder is mounted inside an app:
// a Moodle plugin (PluginType and PluginName) for a Moodle app, or a
// folder relative to the app's project root (RelPath) for the other app
// types.
type DevFolderTarget struct {
	PluginType string
	PluginName string
	RelPath    string
}

// targetOf returns the DevFolderTarget a recorded mount was made with.
func targetOf(m config.MoodlePluginMount) DevFolderTarget {
	if m.Component != "" {
		ptype, name, _ := strings.Cut(m.Component, "_")
		return DevFolderTarget{PluginType: ptype, PluginName: name}
	}
	return DevFolderTarget{RelPath: m.RelPath}
}

// PlanDevFolder validates mounting dev into the app in folder at t: a
// Moodle app takes a plugin (PlanMoodlePlugin), any other app a folder
// relative to its project root. In a PHP + Node app that folder must be
// inside api/ or app/, the two halves its container sees.
func PlanDevFolder(cfg *config.Config, dev, folder string, t DevFolderTarget) (MoodlePluginPlan, error) {
	app, found := FindAppByFolder(cfg.Docker.Apps, folder)
	if !found {
		return MoodlePluginPlan{}, errors.New(i18n.Tr("devfolder.noApp", i18n.Args{"app": folder}))
	}
	if app.Type == config.AppTypeMoodle {
		return PlanMoodlePlugin(cfg, dev, folder, t.PluginType, t.PluginName)
	}
	workspace, err := resolveWorkspace(cfg)
	if err != nil {
		return MoodlePluginPlan{}, err
	}
	if strings.TrimSpace(t.RelPath) == "" {
		return MoodlePluginPlan{}, i18n.NewError("devfolder.noDest")
	}
	relPath := filepath.Clean(strings.TrimSpace(t.RelPath))
	if !filepath.IsLocal(relPath) || relPath == "." {
		return MoodlePluginPlan{}, errors.New(i18n.Tr("devfolder.destOutside", i18n.Args{"app": folder}))
	}
	if _, ok := containerPath(app.Type, relPath); !ok {
		return MoodlePluginPlan{}, i18n.NewError("devfolder.comboHalf")
	}
	return planMount(cfg, workspace, dev, folder, relPath, "")
}

// containerPath returns where relPath, relative to the project root of an
// app of appType on the host, appears inside the app's container: under
// /var/www/html for the PHP types, under /app for Node, and under either
// for PHP + Node (whose api/ and app/ halves are mounted apart). ok is
// false for a PHP + Node path outside both halves.
func containerPath(appType, relPath string) (string, bool) {
	rel := filepath.ToSlash(relPath)
	switch appType {
	case config.AppTypeNode:
		return "/app/" + rel, true
	case config.AppTypePHPNode:
		if rest, ok := strings.CutPrefix(rel, "api/"); ok && rest != "" {
			return "/var/www/html/" + rest, true
		}
		if rest, ok := strings.CutPrefix(rel, "app/"); ok && rest != "" {
			return "/app/" + rest, true
		}
		return "", false
	default:
		return "/var/www/html/" + rel, true
	}
}

// nginxServes reports whether Nginx serves the files of an app of appType
// (every type but Node, which Nginx only proxies), so a mount into it must
// reach Nginx too.
func nginxServes(appType string) bool {
	return appType != config.AppTypeNode
}

// planMount validates mounting dev at relPath inside the project root of
// the app in folder: dev must be an existing folder outside the app's
// code, the destination must stay inside the project root and not exist
// yet or be an empty folder, and no other record may use it.
func planMount(cfg *config.Config, workspace, dev, folder, relPath, component string) (MoodlePluginPlan, error) {
	devPath, err := resolveExistingDir(dev)
	if err != nil {
		return MoodlePluginPlan{}, err
	}
	html, err := resolveExistingDir(AppHTMLDir(workspace, folder))
	if err != nil {
		return MoodlePluginPlan{}, err
	}
	dest := filepath.Join(html, relPath)
	if !resolvedWithin(html, dest) {
		return MoodlePluginPlan{}, errors.New(i18n.Tr("devfolder.destOutsideAt", i18n.Args{"dest": dest, "app": folder}))
	}
	if within(html, devPath) || within(devPath, html) {
		return MoodlePluginPlan{}, errors.New(i18n.Tr("devfolder.devInsideApp", i18n.Args{"app": folder}))
	}
	for _, p := range []string{devPath, dest} {
		if strings.ContainsFunc(p, func(r rune) bool { return r <= ' ' || r == 0x7f || r == '\\' }) {
			return MoodlePluginPlan{}, errors.New(i18n.Tr("devfolder.badPath", i18n.Args{"path": p}))
		}
	}
	for _, m := range cfg.MoodlePlugins {
		if m.App == folder && filepath.Clean(m.RelPath) == relPath {
			return MoodlePluginPlan{}, errors.New(i18n.Tr("devfolder.taken", i18n.Args{"dest": filepath.Join(folder, relPath)}))
		}
	}
	if err := requireEmptyOrMissing(dest); err != nil {
		return MoodlePluginPlan{}, err
	}
	return MoodlePluginPlan{
		App: folder, Component: component, DevPath: devPath, Dest: dest, RelPath: relPath, HTML: html,
	}, nil
}

// label is how messages name the mount: the plugin's component, or the
// destination inside the app.
func (p MoodlePluginPlan) label() string {
	if p.Component != "" {
		return p.Component
	}
	return filepath.Join(p.App, p.RelPath)
}

// mountLabel is MoodlePluginPlan.label for a recorded mount.
func mountLabel(m config.MoodlePluginMount) string {
	if m.Component != "" {
		return m.Component
	}
	return filepath.Join(m.App, m.RelPath)
}

// requireEmptyOrMissing refuses a destination that already has content
// (or isn't a plain folder).
func requireEmptyOrMissing(dest string) error {
	info, err := os.Lstat(dest)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New(i18n.Tr("devfolder.fileAtDest", i18n.Args{"dest": dest}))
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return errors.New(i18n.Tr("devfolder.destNotEmpty", i18n.Args{"dest": dest}))
	}
	return nil
}

// AddMoodlePlugin mounts dev as the plugin ptype_name of the Moodle app in
// folder (AddDevFolder).
func AddMoodlePlugin(ctx context.Context, exe *executor.Executor, stdout io.Writer, dev, folder, ptype, name string) error {
	return AddDevFolder(ctx, exe, stdout, dev, folder, DevFolderTarget{PluginType: ptype, PluginName: name})
}

// AddDevFolder creates the empty destination folder, records the mount,
// then installs and starts the mount unit (one password prompt). When the
// unit cannot be installed or started, the privileged batch removes it
// again and the record and the new folder are dropped. Containers pick the
// mount up on their next restart. A destination inside a MEGA-synced folder
// is first excluded in that sync's .megaignore; the mount is refused when
// the exclusion cannot be written.
func AddDevFolder(ctx context.Context, exe *executor.Executor, stdout io.Writer, dev, folder string, t DevFolderTarget) error {
	stackMu.Lock()
	defer stackMu.Unlock()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	plan, err := PlanDevFolder(cfg, dev, folder, t)
	if err != nil {
		return err
	}
	p, err := prepareMount(stdout, plan)
	if err != nil {
		return err
	}
	kept, err := installPrepared(ctx, exe, stdout, []preparedMount{p})
	if err != nil {
		// A mount the rollback could not stop keeps its unit, so the record
		// is kept too: Remover can still undo it.
		if len(kept) > 0 {
			return fmt.Errorf("%w — %s", err, i18n.Tr("devfolder.keptActive", nil))
		}
		return err
	}
	ui.Success(stdout, i18n.Tr("devfolder.run.mounted", i18n.Args{"name": plan.label(), "dest": plan.Dest}))
	return nil
}

// preparedMount is a planned mount whose MEGA exclusion and destination
// folder are in place, waiting for its mount unit.
type preparedMount struct {
	plan    MoodlePluginPlan
	record  config.MoodlePluginMount
	created bool // the destination folder did not exist before prepareMount
}

// drop removes what prepareMount created: the destination folder (when it
// created it) and the MEGA exclusion.
func (p preparedMount) drop() {
	if p.created {
		_ = os.Remove(p.plan.Dest)
	}
	if p.record.MegaIgnoreRule != "" {
		_ = removeMegaIgnoreRule(p.record.MegaIgnore, p.record.MegaIgnoreRule)
	}
}

// prepareMount writes the MEGA exclusion of plan.Dest (before the folder
// exists, so MEGAsync never scans the folder that is about to be mounted)
// and creates the destination folder. Nothing is left behind on error.
func prepareMount(stdout io.Writer, plan MoodlePluginPlan) (preparedMount, error) {
	megaFile, megaRule, err := excludeFromMega(stdout, plan.Dest)
	if err != nil {
		return preparedMount{}, err
	}
	_, statErr := os.Lstat(plan.Dest)
	p := preparedMount{
		plan: plan,
		record: config.MoodlePluginMount{
			App: plan.App, Component: plan.Component, DevPath: plan.DevPath, RelPath: plan.RelPath,
			MegaIgnore: megaFile, MegaIgnoreRule: megaRule,
		},
		created: errors.Is(statErr, fs.ErrNotExist),
	}
	if err := os.MkdirAll(plan.Dest, 0o755); err != nil {
		p.drop()
		return preparedMount{}, fmt.Errorf("create %s: %w", plan.Dest, err)
	}
	// Checked again right before the privileged step: the folders created
	// above must not have led outside the app through a link.
	if !resolvedWithin(plan.HTML, plan.Dest) {
		p.drop()
		return preparedMount{}, errors.New(i18n.Tr("devfolder.destOutsideAt", i18n.Args{"dest": plan.Dest, "app": plan.App}))
	}
	return p, nil
}

// installPrepared records every prepared mount, then installs and starts
// all their mount units in one privileged batch (one password prompt).
// When the batch fails, the mounts that are active anyway are kept with
// their records; the others lose their record, folder and MEGA exclusion.
// It returns the mounts kept and the batch error.
func installPrepared(ctx context.Context, exe *executor.Executor, stdout io.Writer, prepared []preparedMount) ([]preparedMount, error) {
	if len(prepared) == 0 {
		return nil, nil
	}
	dropAll := func(list []preparedMount) {
		for _, p := range list {
			p.drop()
		}
	}
	if err := config.Update(func(c *config.Config) error {
		for _, p := range prepared {
			c.MoodlePlugins = append(c.MoodlePlugins, p.record)
		}
		return nil
	}); err != nil {
		dropAll(prepared)
		return nil, err
	}

	var steps []executor.PrivilegedStep
	for _, p := range prepared {
		unitSteps, tmp, err := mountUnitSteps(p.plan)
		if tmp != "" {
			defer func() { _ = os.Remove(tmp) }()
		}
		if err != nil {
			return dropFailed(prepared, fmt.Errorf("prepare the mount unit: %w", err))
		}
		steps = append(steps, unitSteps...)
	}
	ui.Info(stdout, i18n.Tr("devfolder.run.authMount", nil))
	if err := exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps); err != nil {
		return dropFailed(prepared, fmt.Errorf("mount: %w", err))
	}
	return prepared, nil
}

// dropFailed keeps the prepared mounts that are active anyway and removes
// the record, folder and MEGA exclusion of the others. It returns the
// mounts kept and err.
func dropFailed(prepared []preparedMount, err error) ([]preparedMount, error) {
	var kept, dropped []preparedMount
	for _, p := range prepared {
		if mountActive(p.plan.DevPath, p.plan.Dest) {
			kept = append(kept, p)
		} else {
			dropped = append(dropped, p)
		}
	}
	_ = config.Update(func(c *config.Config) error {
		for _, p := range dropped {
			c.MoodlePlugins = withoutMoodlePlugin(c.MoodlePlugins, p.plan.App, p.plan.RelPath)
		}
		return nil
	})
	for _, p := range dropped {
		p.drop()
	}
	return kept, err
}

// mountActive reports whether dev is mounted over dest. Replaced in tests.
var mountActive = sameDir

// withoutMoodlePlugin returns mounts minus the record for (app, relPath).
func withoutMoodlePlugin(mounts []config.MoodlePluginMount, app, relPath string) []config.MoodlePluginMount {
	kept := make([]config.MoodlePluginMount, 0, len(mounts))
	for _, r := range mounts {
		if r.App != app || filepath.Clean(r.RelPath) != filepath.Clean(relPath) {
			kept = append(kept, r)
		}
	}
	return kept
}

// guardedRemoveScript removes the unit file $2 unless the folder $3 is
// still a mount point: a unit that failed to stop is kept, so the mount is
// not left active with nothing recording how to stop it.
const guardedRemoveScript = `if mountpoint -q -- "$3"; then printf '%s\n' "$3 is still mounted — the unit was kept." >&2; exit 1; fi; rm -f -- "$2"`

// enableOrRollbackScript enables and starts the unit $1; when that fails it
// stops and removes it again (unit file $2, mount folder $3), so a failed
// mount leaves nothing behind that would try to mount at boot.
const enableOrRollbackScript = `systemctl enable --now -- "$1" && exit 0; systemctl disable --now -- "$1"; ` +
	guardedRemoveScript + `; systemctl daemon-reload; exit 1`

// unmountSteps stops and disables unit and removes its file (unless dest is
// still mounted), then reloads systemd. Used by RemoveMoodlePlugin.
func unmountSteps(unit, dest string) []executor.PrivilegedStep {
	unitFile := filepath.Join(systemdUnitDir, unit)
	return []executor.PrivilegedStep{
		{Announce: i18n.Tr("devfolder.run.unmounting", i18n.Args{"dest": dest}), Soft: true, WarnMessage: i18n.Tr("devfolder.run.unitGone", i18n.Args{"unit": unit}), Name: "systemctl", Args: []string{"disable", "--now", "--", unit}},
		{Name: "sh", Args: []string{"-c", guardedRemoveScript, "sh", unit, unitFile, dest}},
		{Name: "systemctl", Args: []string{"daemon-reload"}},
	}
}

// mountUnitSteps stages plan's mount unit in a temporary file (returned in
// tmp, for the caller to remove) and returns the privileged steps that
// install it through a root-owned copy and enable it. If enabling fails,
// the steps remove the unit again.
func mountUnitSteps(plan MoodlePluginPlan) (steps []executor.PrivilegedStep, tmp string, err error) {
	content := mountUnitContent(plan)
	f, err := os.CreateTemp("", "perci-mount-*.mount")
	if err != nil {
		return nil, "", err
	}
	tmp = f.Name()
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return nil, tmp, err
	}
	if err := f.Close(); err != nil {
		return nil, tmp, err
	}
	sum := sha256.Sum256([]byte(content))

	unit := mountUnitName(plan.Dest)
	unitFile := filepath.Join(systemdUnitDir, unit)
	steps, err = executor.PrivilegedInstallSteps(tmp, unitFile, hex.EncodeToString(sum[:]), "0644")
	if err != nil {
		return nil, tmp, err
	}
	steps[0].Announce = i18n.Tr("devfolder.run.installUnit", i18n.Args{"unit": unit})
	steps = append(steps,
		executor.PrivilegedStep{Name: "systemctl", Args: []string{"daemon-reload"}},
		executor.PrivilegedStep{Announce: i18n.Tr("devfolder.run.mounting", i18n.Args{"dest": plan.Dest}), Name: "sh", Args: []string{"-c", enableOrRollbackScript, "sh", unit, unitFile, plan.Dest}},
	)
	return steps, tmp, nil
}

// mountUnitHeader is the first line of every mount unit perci installs; it
// marks the units RemoveDevFolder may remove, so it never changes.
const mountUnitHeader = "# Generated by perci.gnl: Dev Tools > Plugins Moodle."

// unitPath escapes a path for use as a systemd unit setting value, where
// '%' starts a specifier.
func unitPath(p string) string { return strings.ReplaceAll(p, "%", "%%") }

// mountUnitContent is the systemd mount unit binding plan.DevPath over
// plan.Dest, ordered before docker.service.
func mountUnitContent(plan MoodlePluginPlan) string {
	return mountUnitHeader + "\n" +
		"[Unit]\n" +
		"Description=perci: " + plan.label() + " in " + plan.App + "\n" +
		"RequiresMountsFor=" + unitPath(plan.DevPath) + "\n" +
		"Before=docker.service\n" +
		"\n" +
		"[Mount]\n" +
		"What=" + unitPath(plan.DevPath) + "\n" +
		"Where=" + unitPath(plan.Dest) + "\n" +
		"Type=none\n" +
		"Options=bind\n" +
		"\n" +
		"[Install]\n" +
		"WantedBy=multi-user.target\n"
}

// mountUnitName is the systemd unit name for a mount at path, escaped the
// way systemd-escape --path --suffix=mount does.
func mountUnitName(path string) string {
	path = strings.Trim(filepath.Clean(path), "/")
	if path == "" {
		return "-.mount"
	}
	var b strings.Builder
	for i := 0; i < len(path); i++ {
		c := path[i]
		switch {
		case c == '/':
			b.WriteByte('-')
		case c == '.' && i == 0:
			fmt.Fprintf(&b, `\x%02x`, c)
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == ':', c == '_', c == '.':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, `\x%02x`, c)
		}
	}
	return b.String() + ".mount"
}

// findMoodlePlugin returns the record for (folder, relPath).
func findMoodlePlugin(mounts []config.MoodlePluginMount, folder, relPath string) (config.MoodlePluginMount, bool) {
	for _, m := range mounts {
		if m.App == folder && filepath.Clean(m.RelPath) == filepath.Clean(relPath) {
			return m, true
		}
	}
	return config.MoodlePluginMount{}, false
}

// pluginUnit is an installed mount unit of a plugin: its name and the
// folder it mounts over.
type pluginUnit struct {
	name  string
	where string
}

// pluginUnits lists the mount units that mount m: the perci-generated units
// in systemdUnitDir whose What= is m.DevPath and whose Where= is dest (or
// ends with <app>/<relPath>, should the workspace path have changed since),
// plus the unit named after dest itself. Looking at the installed units
// keeps the removal working when the unit's name can no longer be derived
// from the current paths.
func pluginUnits(m config.MoodlePluginMount, dest string) []pluginUnit {
	units := []pluginUnit{{name: mountUnitName(dest), where: dest}}
	entries, err := os.ReadDir(systemdUnitDir)
	if err != nil {
		return units
	}
	suffix := "/" + m.App + "/" + filepath.ToSlash(filepath.Clean(m.RelPath))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".mount") || e.Name() == units[0].name {
			continue
		}
		data, err := os.ReadFile(filepath.Join(systemdUnitDir, e.Name()))
		if err != nil || !strings.HasPrefix(string(data), mountUnitHeader) {
			continue
		}
		var what, where string
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := strings.CutPrefix(line, "What="); ok {
				what = strings.ReplaceAll(v, "%%", "%")
			}
			if v, ok := strings.CutPrefix(line, "Where="); ok {
				where = strings.ReplaceAll(v, "%%", "%")
			}
		}
		if what == m.DevPath && where != "" && (where == dest || strings.HasSuffix(where, suffix)) {
			units = append(units, pluginUnit{name: e.Name(), where: where})
		}
	}
	return units
}

// RemoveDevFolder stops and removes the mount unit of a recorded mount
// (one password prompt), removes the destination folder only when it is
// left empty and unmounted, drops the record and restarts the containers
// so they stop seeing the folder. A .megaignore rule perci added for the
// mount is removed after the folder is unmounted. A unit whose mount could
// not be stopped is kept, and the record with it. The development folder is
// never touched.
func RemoveDevFolder(ctx context.Context, exe *executor.Executor, stdout io.Writer, folder, relPath string) error {
	stackMu.Lock()
	defer stackMu.Unlock()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	m, found := findMoodlePlugin(cfg.MoodlePlugins, folder, relPath)
	if !found {
		return errors.New(i18n.Tr("devfolder.notMounted", i18n.Args{"dest": filepath.Join(folder, relPath)}))
	}
	workspace, err := resolveWorkspace(cfg)
	if err != nil {
		return err
	}
	html := AppHTMLDir(workspace, folder)
	if resolved, err := filepath.EvalSymlinks(html); err == nil {
		html = resolved
	}
	// The recorded path comes from config.yaml: it is checked before it
	// reaches the privileged step.
	if !filepath.IsLocal(m.RelPath) {
		return fmt.Errorf("the mount record points outside the app: %s", m.RelPath)
	}
	dest := filepath.Join(html, m.RelPath)
	if !resolvedWithin(html, dest) {
		return errors.New(i18n.Tr("devfolder.destOutsideAt", i18n.Args{"dest": dest, "app": folder}))
	}

	ui.Info(stdout, i18n.Tr("devfolder.run.authUnmount", nil))
	var steps []executor.PrivilegedStep
	for _, u := range pluginUnits(m, dest) {
		steps = append(steps, unmountSteps(u.name, u.where)...)
	}
	if err := exe.RunSudoSequence(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, steps); err != nil {
		return fmt.Errorf("unmount: %w", err)
	}

	if !exe.DryRun {
		if sameDir(m.DevPath, dest) {
			return errors.New(i18n.Tr("devfolder.stillMounted", i18n.Args{"dest": dest}))
		}
		if err := os.Remove(dest); err != nil && !errors.Is(err, fs.ErrNotExist) {
			ui.Warning(stdout, i18n.Tr("devfolder.run.destKept", i18n.Args{"dest": dest, "error": err.Error()}))
		}
	}
	if m.MegaIgnoreRule != "" {
		if err := removeMegaIgnoreRule(m.MegaIgnore, m.MegaIgnoreRule); err != nil {
			ui.Warning(stdout, i18n.Tr("devfolder.run.megaRuleKept", i18n.Args{"rule": m.MegaIgnoreRule, "path": m.MegaIgnore, "error": err.Error()}))
		}
	}
	if err := config.Update(func(c *config.Config) error {
		c.MoodlePlugins = withoutMoodlePlugin(c.MoodlePlugins, m.App, m.RelPath)
		return nil
	}); err != nil {
		return err
	}
	ui.Success(stdout, i18n.Tr("devfolder.run.unmounted", i18n.Args{"name": mountLabel(m), "app": folder}))
	// An app no longer registered may still have a container serving the
	// folder through Nginx.
	app, found := FindAppByFolder(cfg.Docker.Apps, folder)
	return restartForPlugins(ctx, exe, stdout, folder, !found || nginxServes(app.Type))
}

// ApplyDevFolders restarts the app container (when running) and, for an
// app Nginx serves, Nginx (when running), so they see the current mounts.
func ApplyDevFolders(ctx context.Context, exe *executor.Executor, stdout io.Writer, folder string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	app, found := FindAppByFolder(cfg.Docker.Apps, folder)
	if !found {
		return errors.New(i18n.Tr("devfolder.noApp", i18n.Args{"app": folder}))
	}
	return restartForPlugins(ctx, exe, stdout, folder, nginxServes(app.Type))
}

// restartForPlugins restarts the app container in folder when it runs, and
// Nginx when withNginx and it runs.
func restartForPlugins(ctx context.Context, exe *executor.Executor, stdout io.Writer, folder string, withNginx bool) error {
	if ContainerStatus(ctx, exe, folder) == "running" {
		ui.Info(stdout, i18n.Tr("devfolder.run.restarting", i18n.Args{"name": folder}))
		if err := RestartContainer(ctx, exe, stdout, folder); err != nil {
			return err
		}
	} else {
		ui.Info(stdout, i18n.Tr("devfolder.run.notRunning", i18n.Args{"name": folder}))
	}
	if withNginx && nginxContainerRunning(ctx, exe) {
		ui.Info(stdout, i18n.Tr("devfolder.run.restartingNginx", nil))
		if err := RestartContainer(ctx, exe, stdout, NginxContainerName); err != nil {
			return err
		}
	}
	ui.Success(stdout, i18n.Tr("devfolder.run.restarted", nil))
	return nil
}

// MoodlePluginState is a recorded mount with its current status and, for a
// Moodle plugin whose dev/version.php declares a different component, that
// component.
type MoodlePluginState struct {
	config.MoodlePluginMount
	Status     string
	DeclaredAs string
}

// MoodlePluginStatuses checks every recorded mount, sorted by component
// and then by app. The checks compare the device and inode of the
// development folder with those of the destination on the host, inside
// the app container and inside Nginx: a bind mount shows the same pair on
// every side, even for an empty folder. Each container is inspected once
// and asked once for all the folders it must hold.
func MoodlePluginStatuses(ctx context.Context, exe *executor.Executor, cfg *config.Config) []MoodlePluginState {
	if len(cfg.MoodlePlugins) == 0 {
		return nil
	}
	workspace, wsErr := resolveWorkspace(cfg)
	nginxUp := nginxContainerRunning(ctx, exe)
	appTypes := map[string]string{}
	for _, a := range cfg.Docker.Apps {
		appTypes[a.Folder] = a.Type
	}

	// Per app: the container status and, for a running one, the ids of the
	// mounted folders as the container sees them.
	appStatus := map[string]string{}
	appPaths := map[string][]string{}
	nginxPaths := []string{}
	for _, m := range cfg.MoodlePlugins {
		if _, seen := appStatus[m.App]; !seen {
			appStatus[m.App] = ContainerStatus(ctx, exe, m.App)
		}
		if p, ok := containerPath(appTypes[m.App], m.RelPath); ok && appStatus[m.App] == "running" {
			appPaths[m.App] = append(appPaths[m.App], p)
		}
		nginxPaths = append(nginxPaths, appHTMLMount+"/"+m.App+"/"+m.RelPath)
	}
	appIDs := map[string]map[string]string{}
	for app, paths := range appPaths {
		appIDs[app] = containerDirIDs(ctx, exe, app, paths)
	}
	var nginxIDs map[string]string
	if nginxUp {
		nginxIDs = containerDirIDs(ctx, exe, NginxContainerName, nginxPaths)
	}

	states := make([]MoodlePluginState, 0, len(cfg.MoodlePlugins))
	for _, m := range cfg.MoodlePlugins {
		st := MoodlePluginState{MoodlePluginMount: m}
		if t, n := DevPluginComponent(m.DevPath); m.Component != "" && t != "" && t+"_"+n != m.Component {
			st.DeclaredAs = t + "_" + n
		}
		appType := appTypes[m.App]
		st.Status = pluginStatus(workspace, wsErr, m, appType, appStatus[m.App], appIDs[m.App], nginxUp && nginxServes(appType), nginxIDs)
		states = append(states, st)
	}
	sort.SliceStable(states, func(i, j int) bool {
		if states[i].Component != states[j].Component {
			return states[i].Component < states[j].Component
		}
		return states[i].App < states[j].App
	})
	return states
}

// pluginStatus classifies one mount into an app of appType from the host's
// view of it, the status of its app container, and the folder ids the app
// container and Nginx reported; nginxUp is false when Nginx doesn't run or
// doesn't serve the app.
func pluginStatus(workspace string, wsErr error, m config.MoodlePluginMount, appType, appStatus string, appIDs map[string]string, nginxUp bool, nginxIDs map[string]string) string {
	devID, ok := dirID(m.DevPath)
	if wsErr != nil || !ok {
		return PluginInactive
	}
	hostID, ok := dirID(filepath.Join(AppHTMLDir(workspace, m.App), m.RelPath))
	if !ok || hostID != devID {
		return PluginInactive
	}
	switch appStatus {
	case "":
		return PluginNoApp
	case "running":
	default:
		return PluginStopped
	}
	if p, ok := containerPath(appType, m.RelPath); !ok || appIDs[p] != devID {
		return PluginPending
	}
	if nginxUp && nginxIDs[appHTMLMount+"/"+m.App+"/"+m.RelPath] != devID {
		return PluginPending
	}
	return PluginApplied
}

// dirID returns "device:inode" of an existing folder.
func dirID(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%d:%d", st.Dev, st.Ino), true
}

// containerDirIDs returns "device:inode" for each of paths inside a running
// container, from one `docker exec ... stat`. A path it cannot read is
// absent from the result.
func containerDirIDs(ctx context.Context, exe *executor.Executor, container string, paths []string) map[string]string {
	ids := map[string]string{}
	if len(paths) == 0 {
		return ids
	}
	// stat exits non-zero when any path is missing but still prints the
	// others; the script swallows that status so the output is kept.
	args := append([]string{"exec", container, "sh", "-c", `stat -c '%n %d:%i' -- "$@" 2>/dev/null || true`, "sh"}, paths...)
	out, _ := exe.Output(ctx, executor.Options{}, "docker", args...)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if i := strings.LastIndex(line, " "); i > 0 {
			ids[line[:i]] = line[i+1:]
		}
	}
	return ids
}

// sameDir reports whether a and b are the same existing folder.
func sameDir(a, b string) bool {
	ida, ok := dirID(a)
	if !ok {
		return false
	}
	idb, ok := dirID(b)
	return ok && ida == idb
}

// resolveExistingDir returns dir's absolute path with links resolved, or
// an error when it is not an existing folder.
func resolveExistingDir(dir string) (string, error) {
	if !filepath.IsAbs(dir) {
		return "", errors.New(i18n.Tr("devfolder.badDir", i18n.Args{"path": dir}))
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", errors.New(i18n.Tr("devfolder.dirNotFound", i18n.Args{"path": dir}))
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New(i18n.Tr("devfolder.notADir", i18n.Args{"path": dir}))
	}
	return resolved, nil
}

// resolvedWithin resolves the deepest existing ancestor of path, following
// links, and reports whether the final location stays under root. An
// ancestor that exists but cannot be resolved (a dangling link) counts as
// outside.
func resolvedWithin(root, path string) bool {
	if resolvedRoot, err := filepath.EvalSymlinks(root); err == nil {
		root = resolvedRoot
	}
	for p := path; ; p = filepath.Dir(p) {
		if _, err := os.Lstat(p); err == nil {
			real, err := filepath.EvalSymlinks(p)
			if err != nil {
				return false
			}
			rest, err := filepath.Rel(p, path)
			if err != nil {
				return false
			}
			return within(root, filepath.Join(real, rest))
		}
		if filepath.Dir(p) == p {
			return false
		}
	}
}

// within reports whether path is root or below it.
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
