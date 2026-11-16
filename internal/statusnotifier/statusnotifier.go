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

// Package statusnotifier shows a system tray icon through the
// StatusNotifierItem D-Bus protocol, with its menu exported through
// com.canonical.dbusmenu. Each Item owns a private session bus connection:
// closing it removes the icon from the panel at once.
package statusnotifier

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

// D-Bus names, paths and interfaces of the protocol.
const (
	itemPath      = dbus.ObjectPath("/StatusNotifierItem")
	menuPath      = dbus.ObjectPath("/MenuBar")
	itemIface     = "org.kde.StatusNotifierItem"
	menuIface     = "com.canonical.dbusmenu"
	watcherName   = "org.kde.StatusNotifierWatcher"
	watcherPath   = dbus.ObjectPath("/StatusNotifierWatcher")
	watcherMethod = watcherName + ".RegisterStatusNotifierItem"
)

// registerTimeout bounds each registration call to the watcher.
const registerTimeout = 5 * time.Second

// Pixmap is one ARGB32 image in network byte order, the D-Bus
// representation (iiay) of the protocol's icon pixmaps.
type Pixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

// PixmapFromPNG decodes a PNG into a Pixmap.
func PixmapFromPNG(data []byte) (Pixmap, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return Pixmap{}, fmt.Errorf("decode tray icon: %w", err)
	}
	b := img.Bounds()
	rgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	out := make([]byte, 0, len(rgba.Pix))
	for i := 0; i < len(rgba.Pix); i += 4 {
		out = append(out, rgba.Pix[i+3], rgba.Pix[i], rgba.Pix[i+1], rgba.Pix[i+2])
	}
	return Pixmap{Width: int32(b.Dx()), Height: int32(b.Dy()), Data: out}, nil
}

// MenuItem is one entry of the tray menu: a clickable label, or a
// separator line when Separator is set.
type MenuItem struct {
	Label     string
	Separator bool
	// OnClick runs when the entry is clicked, on the D-Bus dispatch
	// goroutine; it must return quickly.
	OnClick func()
}

// Options describes the tray icon.
type Options struct {
	// ID identifies the application to the panel.
	ID string
	// Title is the icon's name and tooltip.
	Title string
	// IconName is an icon looked up in the desktop's icon theme; a name
	// ending in "-symbolic" is painted by the panel in its own text color.
	IconName string
	// Pixmap is shown when the panel finds no icon named IconName.
	Pixmap *Pixmap
	// OnActivate runs on the icon's activation (a primary click on most
	// panels, a double click on GNOME's), on the D-Bus dispatch goroutine;
	// it must return quickly.
	OnActivate func()
	// Menu is the context menu's content.
	Menu []MenuItem
}

// tooltip is the D-Bus representation (sa(iiay)ss) of the ToolTip
// property.
type tooltip struct {
	IconName    string
	Pixmaps     []Pixmap
	Title       string
	Description string
}

// itemSeq numbers the bus names requested by this process.
var itemSeq atomic.Uint32

// Item is a tray icon shown on the panel until Close.
type Item struct {
	conn      *dbus.Conn
	itemProps *prop.Properties
	menuProps *prop.Properties
	menu      *menuModel
	busName   string
	done      chan struct{}
	closeOnce sync.Once
}

// Start connects to the session bus, exports the icon and its menu, and
// registers them with the panel's StatusNotifierWatcher — again whenever a
// new watcher appears. A missing watcher is not an error: the icon shows
// once one starts.
func Start(opts Options) (*Item, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to the session bus: %w", err)
	}
	it := &Item{
		conn:    conn,
		menu:    newMenuModel(opts.Menu),
		busName: fmt.Sprintf("org.kde.StatusNotifierItem-%d-%d", os.Getpid(), itemSeq.Add(1)),
		done:    make(chan struct{}),
	}
	if err := it.export(opts); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err := conn.RequestName(it.busName, dbus.NameFlagDoNotQueue); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("request bus name %s: %w", it.busName, err)
	}
	if err := conn.AddMatchSignal(
		dbus.WithMatchSender("org.freedesktop.DBus"),
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchArg(0, watcherName),
	); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("watch for the tray watcher: %w", err)
	}
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	_ = it.register()
	go it.watch(signals)
	return it, nil
}

// export publishes the item and menu objects, their properties and their
// introspection data.
func (it *Item) export(opts Options) error {
	item := &itemObject{onActivate: opts.OnActivate}
	if err := it.conn.Export(item, itemPath, itemIface); err != nil {
		return fmt.Errorf("export tray item: %w", err)
	}
	menu := &menuObject{model: it.menu}
	if err := it.conn.Export(menu, menuPath, menuIface); err != nil {
		return fmt.Errorf("export tray menu: %w", err)
	}

	var pixmaps []Pixmap
	if opts.Pixmap != nil {
		pixmaps = []Pixmap{*opts.Pixmap}
	}
	ro := func(v any) *prop.Prop { return &prop.Prop{Value: v, Emit: prop.EmitTrue} }
	itemProps, err := prop.Export(it.conn, itemPath, prop.Map{itemIface: {
		"Category":            ro("ApplicationStatus"),
		"Id":                  ro(opts.ID),
		"Title":               ro(opts.Title),
		"Status":              ro("Active"),
		"WindowId":            ro(int32(0)),
		"IconName":            ro(opts.IconName),
		"IconThemePath":       ro(""),
		"IconPixmap":          ro(pixmaps),
		"OverlayIconName":     ro(""),
		"OverlayIconPixmap":   ro([]Pixmap{}),
		"AttentionIconName":   ro(""),
		"AttentionIconPixmap": ro([]Pixmap{}),
		"AttentionMovieName":  ro(""),
		"ToolTip":             ro(tooltip{IconName: opts.IconName, Pixmaps: []Pixmap{}, Title: opts.Title}),
		"ItemIsMenu":          ro(false),
		"Menu":                ro(menuPath),
	}})
	if err != nil {
		return fmt.Errorf("export tray item properties: %w", err)
	}
	menuProps, err := prop.Export(it.conn, menuPath, prop.Map{menuIface: {
		"Version":       ro(uint32(3)),
		"TextDirection": ro("ltr"),
		"Status":        ro("normal"),
		"IconThemePath": ro([]string{}),
	}})
	if err != nil {
		return fmt.Errorf("export tray menu properties: %w", err)
	}
	it.itemProps = itemProps
	it.menuProps = menuProps

	itemNode := &introspect.Node{
		Name: string(itemPath),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       itemIface,
				Methods:    introspect.Methods(item),
				Properties: itemProps.Introspection(itemIface),
				Signals: []introspect.Signal{
					{Name: "NewTitle"}, {Name: "NewIcon"}, {Name: "NewAttentionIcon"},
					{Name: "NewOverlayIcon"}, {Name: "NewToolTip"},
					{Name: "NewStatus", Args: []introspect.Arg{{Name: "status", Type: "s"}}},
				},
			},
		},
	}
	if err := it.conn.Export(introspect.NewIntrospectable(itemNode), itemPath,
		"org.freedesktop.DBus.Introspectable"); err != nil {
		return fmt.Errorf("export tray item introspection: %w", err)
	}
	menuNode := &introspect.Node{
		Name: string(menuPath),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       menuIface,
				Methods:    introspect.Methods(menu),
				Properties: menuProps.Introspection(menuIface),
				Signals: []introspect.Signal{
					{Name: "LayoutUpdated", Args: []introspect.Arg{
						{Name: "revision", Type: "u"}, {Name: "parent", Type: "i"},
					}},
					{Name: "ItemsPropertiesUpdated", Args: []introspect.Arg{
						{Name: "updatedProps", Type: "a(ia{sv})"}, {Name: "removedProps", Type: "a(ias)"},
					}},
					{Name: "ItemActivationRequested", Args: []introspect.Arg{
						{Name: "id", Type: "i"}, {Name: "timestamp", Type: "u"},
					}},
				},
			},
		},
	}
	if err := it.conn.Export(introspect.NewIntrospectable(menuNode), menuPath,
		"org.freedesktop.DBus.Introspectable"); err != nil {
		return fmt.Errorf("export tray menu introspection: %w", err)
	}
	return nil
}

// register announces the item to the current StatusNotifierWatcher.
func (it *Item) register() error {
	ctx, cancel := context.WithTimeout(context.Background(), registerTimeout)
	defer cancel()
	call := it.conn.Object(watcherName, watcherPath).CallWithContext(ctx, watcherMethod, 0, it.busName)
	if call.Err != nil {
		return fmt.Errorf("register the tray icon: %w", call.Err)
	}
	return nil
}

// watch registers the item again each time a StatusNotifierWatcher takes
// the watcher name (e.g. the panel restarted), until Close.
func (it *Item) watch(signals <-chan *dbus.Signal) {
	for {
		select {
		case <-it.done:
			return
		case sig, ok := <-signals:
			if !ok || sig == nil {
				return
			}
			if sig.Name != "org.freedesktop.DBus.NameOwnerChanged" || len(sig.Body) < 3 {
				continue
			}
			if name, _ := sig.Body[0].(string); name != watcherName {
				continue
			}
			if owner, _ := sig.Body[2].(string); owner != "" {
				_ = it.register()
			}
		}
	}
}

// SetMenu replaces the menu's content and tells the panel to reload it.
func (it *Item) SetMenu(items []MenuItem) {
	revision := it.menu.set(items)
	_ = it.conn.Emit(menuPath, menuIface+".LayoutUpdated", revision, int32(0))
}

// SetPixmap replaces the fallback pixmap and tells the panel to reload
// the icon.
func (it *Item) SetPixmap(p Pixmap) {
	it.itemProps.SetMust(itemIface, "IconPixmap", []Pixmap{p})
	_ = it.conn.Emit(itemPath, itemIface+".NewIcon")
}

// Close removes the icon from the panel by closing the bus connection.
// Further calls do nothing.
func (it *Item) Close() error {
	var err error
	it.closeOnce.Do(func() {
		close(it.done)
		err = it.conn.Close()
	})
	return err
}

// itemObject implements the org.kde.StatusNotifierItem methods.
type itemObject struct {
	onActivate func()
}

// Activate runs the activation callback.
func (o *itemObject) Activate(_, _ int32) *dbus.Error {
	if o.onActivate != nil {
		o.onActivate()
	}
	return nil
}

// ContextMenu does nothing: panels show the exported menu themselves.
func (o *itemObject) ContextMenu(_, _ int32) *dbus.Error { return nil }

// SecondaryActivate (a middle click) does nothing.
func (o *itemObject) SecondaryActivate(_, _ int32) *dbus.Error { return nil }

// Scroll does nothing.
func (o *itemObject) Scroll(_ int32, _ string) *dbus.Error { return nil }
