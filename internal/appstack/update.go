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

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/executor"
	"github.com/oito2/perci82/internal/i18n"
	"github.com/oito2/perci82/internal/ui"
)

// ImageUpdateTarget names the containers an image update covers: Nginx,
// MariaDB and Containers Aplicativos by folder.
type ImageUpdateTarget struct {
	Nginx   bool
	MariaDB bool
	Apps    []string
}

// ImageUpdatePlan lists the containers whose image changed during
// PrepareImageUpdate and that ApplyImageUpdate recreates.
type ImageUpdatePlan struct {
	Nginx   bool
	MariaDB bool
	Apps    []string
}

// Empty reports whether no container needs to be recreated.
func (p ImageUpdatePlan) Empty() bool {
	return !p.Nginx && !p.MariaDB && len(p.Apps) == 0
}

// AllImagesTarget returns the target covering every container registered
// in config.yaml.
func AllImagesTarget() (ImageUpdateTarget, error) {
	cfg, err := config.Load()
	if err != nil {
		return ImageUpdateTarget{}, fmt.Errorf("load config: %w", err)
	}
	t := ImageUpdateTarget{Nginx: cfg.Docker.NginxCreated, MariaDB: cfg.Docker.MariaDB.DBUser != ""}
	for _, app := range cfg.Docker.Apps {
		t.Apps = append(t.Apps, app.Folder)
	}
	if !t.Nginx && !t.MariaDB && len(t.Apps) == 0 {
		return ImageUpdateTarget{}, i18n.NewError("docker.noContainers")
	}
	return t, nil
}

// AppImageTarget returns the target of the registered app folder together
// with every other registered app built from the same Perci image, since
// they share it.
func AppImageTarget(folder string) (ImageUpdateTarget, error) {
	app, err := registeredApp(folder)
	if err != nil {
		return ImageUpdateTarget{}, err
	}
	cfg, err := config.Load()
	if err != nil {
		return ImageUpdateTarget{}, fmt.Errorf("load config: %w", err)
	}
	name := appImageBuild(app).name
	var t ImageUpdateTarget
	for _, other := range cfg.Docker.Apps {
		if appImageBuild(other).name == name {
			t.Apps = append(t.Apps, other.Folder)
		}
	}
	return t, nil
}

// appImageBuild describes the Perci image app's container runs from.
func appImageBuild(app config.AppContainer) imageBuild {
	switch app.Type {
	case config.AppTypeNode:
		return nodeImageBuild(app.NodeVersion)
	case config.AppTypePHPNode:
		return comboImageBuild(app.PHPVersion, app.NodeVersion)
	default:
		return phpImageBuild(app.PHPVersion)
	}
}

// PrepareImageUpdate pulls the newest build of every upstream image the
// target uses (same tag, e.g. php:8.2-fpm), rebuilds the Perci images
// whose base changed (or that are missing or out of date), and returns the
// existing containers whose image is no longer the current one. No
// container is touched, so a failed pull or build leaves everything
// running as it was.
func PrepareImageUpdate(ctx context.Context, exe *executor.Executor, stdout io.Writer, t ImageUpdateTarget) (ImageUpdatePlan, error) {
	cfg, err := config.Load()
	if err != nil {
		return ImageUpdatePlan{}, fmt.Errorf("load config: %w", err)
	}
	var plan ImageUpdatePlan

	if t.Nginx {
		if !cfg.Docker.NginxCreated {
			return ImageUpdatePlan{}, i18n.NewError("docker.noNginx")
		}
		if _, err := pullImage(ctx, exe, stdout, NginxImage); err != nil {
			return ImageUpdatePlan{}, err
		}
		plan.Nginx = containerOutdated(ctx, exe, stdout, NginxContainerName, NginxImage)
	}
	if t.MariaDB {
		if cfg.Docker.MariaDB.DBUser == "" {
			return ImageUpdatePlan{}, i18n.NewError("docker.noMariaDB")
		}
		if _, err := pullImage(ctx, exe, stdout, MariaDBImage); err != nil {
			return ImageUpdatePlan{}, err
		}
		plan.MariaDB = containerOutdated(ctx, exe, stdout, MariaDBContainerName, MariaDBImage)
	}

	apps := make([]config.AppContainer, 0, len(t.Apps))
	for _, folder := range t.Apps {
		app, err := registeredApp(folder)
		if err != nil {
			return ImageUpdatePlan{}, err
		}
		apps = append(apps, app)
	}
	done := map[string]bool{}
	for _, app := range apps {
		b := appImageBuild(app)
		if done[b.name] {
			continue
		}
		done[b.name] = true
		if err := refreshImage(ctx, exe, stdout, b); err != nil {
			return ImageUpdatePlan{}, err
		}
	}
	for _, app := range apps {
		if containerOutdated(ctx, exe, stdout, app.Folder, appImageBuild(app).name) {
			plan.Apps = append(plan.Apps, app.Folder)
		}
	}
	return plan, nil
}

// refreshImage pulls b's base images and rebuilds b when one of them
// changed, or when b is missing or was built from other content.
func refreshImage(ctx context.Context, exe *executor.Executor, stdout io.Writer, b imageBuild) error {
	changed := false
	for _, base := range b.bases {
		c, err := pullImage(ctx, exe, stdout, base)
		if err != nil {
			return err
		}
		changed = changed || c
	}
	label, err := exe.Output(ctx, executor.Options{}, "docker", "image", "inspect",
		"-f", `{{index .Config.Labels "`+imageLabel+`"}}`, "--", b.name)
	if !changed && err == nil && strings.TrimSpace(label) == b.hash() {
		ui.Info(stdout, i18n.Tr("docker.run.imageUpToDate", i18n.Args{"image": b.name}))
		return nil
	}
	ui.Info(stdout, i18n.Tr("docker.run.imageRebuild", i18n.Args{"image": b.name}))
	return buildImage(ctx, exe, stdout, b)
}

// pullImage runs `docker pull` for ref and reports whether the local image
// changed (a newer build of the same tag was downloaded).
func pullImage(ctx context.Context, exe *executor.Executor, stdout io.Writer, ref string) (changed bool, err error) {
	before := imageID(ctx, exe, ref)
	ui.Info(stdout, i18n.Tr("docker.run.imagePull", i18n.Args{"image": ref}))
	if err := exe.Run(ctx, executor.Options{Stdout: stdout, Stderr: stdout}, "docker", "pull", "--quiet", "--", ref); err != nil {
		return false, fmt.Errorf("pull image %s: %w", ref, err)
	}
	after := imageID(ctx, exe, ref)
	if after != before {
		ui.Info(stdout, i18n.Tr("docker.run.imagePulled", i18n.Args{"image": ref}))
		return true, nil
	}
	ui.Info(stdout, i18n.Tr("docker.run.imageCurrent", i18n.Args{"image": ref}))
	return false, nil
}

// imageID returns the local image ID of ref, or "" when it is absent.
func imageID(ctx context.Context, exe *executor.Executor, ref string) string {
	out, err := exe.Output(ctx, executor.Options{}, "docker", "image", "inspect", "-f", "{{.Id}}", "--", ref)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// containerOutdated reports whether container exists and runs an image
// other than the current local image. A missing container is reported in
// stdout and left alone.
func containerOutdated(ctx context.Context, exe *executor.Executor, stdout io.Writer, container, image string) bool {
	out, err := exe.Output(ctx, executor.Options{}, "docker", "inspect", "-f", "{{.Image}}", "--", container)
	if err != nil {
		ui.Warning(stdout, i18n.Tr("docker.run.containerMissing", i18n.Args{"name": container}))
		return false
	}
	return strings.TrimSpace(out) != imageID(ctx, exe, image)
}

// ApplyImageUpdate recreates the containers of plan from their current
// images, in the order MariaDB, apps, Nginx, then reloads Nginx's routing.
// Recreating keeps every bind-mounted folder (html, data, logs, the
// MariaDB data directory, Moodle plugin mounts on the host). Containers
// that were not running before are stopped again afterwards.
func ApplyImageUpdate(ctx context.Context, exe *executor.Executor, stdout io.Writer, plan ImageUpdatePlan) error {
	if plan.Empty() {
		return nil
	}
	stackMu.Lock()
	defer stackMu.Unlock()

	var stopped []string
	remember := func(name string) {
		if s := ContainerStatus(ctx, exe, name); s != "" && s != "running" {
			stopped = append(stopped, name)
		}
	}

	if plan.MariaDB {
		remember(MariaDBContainerName)
		ui.Info(stdout, i18n.Tr("transfer.run.recreateMariaDB", nil))
		if err := RecreateMariaDB(ctx, exe, stdout); err != nil {
			return fmt.Errorf("recreate mariadb: %w", err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	apps := cfg.Docker.Apps
	for _, folder := range plan.Apps {
		remember(folder)
		ui.Info(stdout, i18n.Tr("transfer.run.recreateApp", i18n.Args{"app": folder}))
		if apps, err = recreateAppContainer(ctx, exe, stdout, folder); err != nil {
			return fmt.Errorf("recreate app %s: %w", folder, err)
		}
	}

	if plan.Nginx {
		remember(NginxContainerName)
		ui.Info(stdout, i18n.Tr("transfer.run.recreateNginx", nil))
		if _, err := recreateNginx(ctx, exe, nil, stdout, false); err != nil {
			return fmt.Errorf("recreate nginx: %w", err)
		}
	}
	if plan.Nginx || len(plan.Apps) > 0 {
		if err := ReloadNginxConfig(ctx, exe, stdout, apps); err != nil {
			return fmt.Errorf("reload nginx: %w", err)
		}
	}

	for _, name := range stopped {
		if err := StopContainer(ctx, exe, stdout, name); err != nil {
			return err
		}
	}
	return nil
}

// MariaDBBackupDir returns ~/.perci/appstack/backups, where the automatic
// dump taken before a MariaDB image update is written, creating it with
// mode 0o700.
func MariaDBBackupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home folder: %w", err)
	}
	dir := filepath.Join(home, ".perci", "appstack", "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}
