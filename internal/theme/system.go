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

package theme

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalBus        = "org.freedesktop.portal.Desktop"
	portalPath       = "/org/freedesktop/portal/desktop"
	portalSettings   = "org.freedesktop.portal.Settings"
	appearance       = "org.freedesktop.appearance"
	gnomeInterface   = "org.gnome.desktop.interface"
	gnomeA11y        = "org.gnome.desktop.a11y.interface"
	readTimeout      = 2 * time.Second
	gsettingsTimeout = 2 * time.Second
)

// portalValues are the appearance keys read from the settings portal;
// nil means the portal didn't answer for that key.
type portalValues struct {
	colorScheme  *uint32
	accent       *[3]float64
	highContrast *uint32
}

// gsettingsGetter returns the value `gsettings get schema key` prints, ""
// when it can't be read.
type gsettingsGetter func(ctx context.Context, schema, key string) string

// compose turns what the portal answered — and, for the keys it didn't
// answer on a GNOME-family desktop (de), GNOME's own settings — into
// SystemSettings. The portal's color-scheme is 1 for dark and 2 for
// light; its accent-color is an RGB triple from 0 to 1 (out of range when
// unset); its contrast is 1 for high contrast.
func compose(ctx context.Context, p portalValues, de string, gsettings gsettingsGetter) SystemSettings {
	var s SystemSettings
	gnome := gnomeFamily(de) && gsettings != nil

	switch {
	case p.colorScheme != nil && *p.colorScheme == 1:
		s.ColorScheme = "dark"
	case p.colorScheme != nil && *p.colorScheme == 2:
		s.ColorScheme = "light"
	case p.colorScheme == nil && gnome:
		switch unquote(gsettings(ctx, gnomeInterface, "color-scheme")) {
		case "prefer-dark":
			s.ColorScheme = "dark"
		case "prefer-light":
			s.ColorScheme = "light"
		}
	}

	if p.accent != nil {
		if a := *p.accent; inUnit(a[0]) && inUnit(a[1]) && inUnit(a[2]) {
			s.Accent = rgb{a[0], a[1], a[2]}.hex()
		}
	}
	if s.Accent == "" && p.accent == nil && gnome {
		if hex, ok := DesignTokens().Accents[unquote(gsettings(ctx, gnomeInterface, "accent-color"))]; ok {
			s.Accent = hex
		}
	}

	s.HighContrast = p.highContrast != nil && *p.highContrast == 1
	if !s.HighContrast && gnome {
		s.HighContrast = unquote(gsettings(ctx, gnomeA11y, "high-contrast")) == "true"
	}
	return s
}

// inUnit reports whether v is within 0 and 1.
func inUnit(v float64) bool { return v >= 0 && v <= 1 }

// unquote strips the quotes gsettings prints around a string value.
func unquote(s string) string {
	return strings.Trim(strings.TrimSpace(s), "'")
}

// ReadSystem reads the desktop's appearance settings: the XDG settings
// portal, and GNOME's settings for what the portal doesn't answer. It
// never fails: what can't be read is left at its zero value.
func ReadSystem(ctx context.Context) SystemSettings {
	var p portalValues
	if conn, err := dbus.ConnectSessionBus(dbus.WithContext(ctx)); err == nil {
		p = readPortal(ctx, conn)
		_ = conn.Close()
	}
	return compose(ctx, p, DetectDesktop(os.Getenv("XDG_CURRENT_DESKTOP"), os.Getenv("DESKTOP_SESSION")), readGsettings)
}

// readPortal reads the appearance keys from the settings portal.
func readPortal(ctx context.Context, conn *dbus.Conn) portalValues {
	var p portalValues
	if v, ok := readPortalKey(ctx, conn, "color-scheme").(uint32); ok {
		p.colorScheme = &v
	}
	if v, ok := readPortalKey(ctx, conn, "contrast").(uint32); ok {
		p.highContrast = &v
	}
	if v, ok := readPortalKey(ctx, conn, "accent-color").([]any); ok && len(v) == 3 {
		var a [3]float64
		valid := true
		for i, c := range v {
			f, isFloat := c.(float64)
			valid = valid && isFloat
			a[i] = f
		}
		if valid {
			p.accent = &a
		}
	}
	return p
}

// readPortalKey returns the value of an appearance key, unwrapped from
// its variants, trying ReadOne and then the older Read; nil when neither
// answers.
func readPortalKey(ctx context.Context, conn *dbus.Conn, key string) any {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	obj := conn.Object(portalBus, portalPath)
	for _, method := range []string{"ReadOne", "Read"} {
		var v dbus.Variant
		if err := obj.CallWithContext(ctx, portalSettings+"."+method, 0, appearance, key).Store(&v); err != nil {
			continue
		}
		value := v.Value()
		for {
			inner, ok := value.(dbus.Variant)
			if !ok {
				break
			}
			value = inner.Value()
		}
		return value
	}
	return nil
}

// readGsettings runs `gsettings get schema key`.
func readGsettings(ctx context.Context, schema, key string) string {
	ctx, cancel := context.WithTimeout(ctx, gsettingsTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gsettings", "get", schema, key).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// WatchSystem calls onChange with the new settings each time the settings
// portal reports a change of an appearance key, until ctx ends. It returns
// an error only when the session bus can't be watched.
func WatchSystem(ctx context.Context, onChange func(SystemSettings)) error {
	conn, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("connect to the session bus: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.AddMatchSignalContext(ctx,
		dbus.WithMatchInterface(portalSettings),
		dbus.WithMatchMember("SettingChanged"),
		dbus.WithMatchArg(0, appearance),
	); err != nil {
		return fmt.Errorf("watch the settings portal: %w", err)
	}
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	last := ReadSystem(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case sig, ok := <-signals:
			if !ok {
				return nil
			}
			if sig == nil || sig.Name != portalSettings+".SettingChanged" {
				continue
			}
			// A theme switch changes several keys at once: the burst is read
			// once, after it ends, instead of once per signal.
			if !settle(ctx, signals) {
				return nil
			}
			if now := ReadSystem(ctx); now != last {
				last = now
				onChange(now)
			}
		}
	}
}

// settleDelay is how long settle waits for another signal before the
// burst counts as over.
var settleDelay = 150 * time.Millisecond

// settle drops the signals that keep arriving on signals, until none
// arrives for settleDelay. It reports false when ctx ended or the channel
// closed.
func settle(ctx context.Context, signals <-chan *dbus.Signal) bool {
	timer := time.NewTimer(settleDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case _, ok := <-signals:
			if !ok {
				return false
			}
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(settleDelay)
		case <-timer.C:
			return true
		}
	}
}
