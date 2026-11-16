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

package statusnotifier

import (
	"bufio"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestPixmapFromPNG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff})
	img.Set(1, 0, color.NRGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0x80})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	got, err := PixmapFromPNG(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0xff, 0x11, 0x22, 0x33, 0x80, 0xaa, 0xbb, 0xcc}
	if got.Width != 2 || got.Height != 1 || !bytes.Equal(got.Data, want) {
		t.Errorf("got %dx%d %x, want 2x1 %x", got.Width, got.Height, got.Data, want)
	}
	if _, err := PixmapFromPNG([]byte("not a png")); err == nil {
		t.Error("got no error for invalid PNG data")
	}
}

func testMenu(clicks *[]string) []MenuItem {
	return []MenuItem{
		{Label: "Open", OnClick: func() { *clicks = append(*clicks, "open") }},
		{Separator: true},
		{Label: "Quit", OnClick: func() { *clicks = append(*clicks, "quit") }},
	}
}

func TestMenuLayout(t *testing.T) {
	var clicks []string
	m := newMenuModel(testMenu(&clicks))

	revision, root, ok := m.layout(rootID)
	if !ok || revision != 1 {
		t.Fatalf("layout(root): ok=%v revision=%d, want true 1", ok, revision)
	}
	if len(root.Children) != 3 {
		t.Fatalf("got %d children, want 3", len(root.Children))
	}
	first := root.Children[0].Value().(layout)
	if first.ID != 1 || first.Props["label"].Value() != "Open" {
		t.Errorf("first child = %+v, want ID 1 labeled Open", first)
	}
	sep := root.Children[1].Value().(layout)
	if sep.Props["type"].Value() != "separator" {
		t.Errorf("second child = %+v, want a separator", sep)
	}

	if _, _, ok := m.layout(9); ok {
		t.Error("layout(9) found a node that doesn't exist")
	}
	if got := m.set(testMenu(&clicks)[:1]); got != 2 {
		t.Errorf("set returned revision %d, want 2", got)
	}
	if _, root, _ := m.layout(rootID); len(root.Children) != 1 {
		t.Errorf("after set: %d children, want 1", len(root.Children))
	}
}

func TestMenuEvents(t *testing.T) {
	var clicks []string
	o := &menuObject{model: newMenuModel(testMenu(&clicks))}

	if err := o.Event(3, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatalf("Event(3, clicked): %v", err)
	}
	if err := o.Event(1, "hovered", dbus.MakeVariant(""), 0); err != nil {
		t.Fatalf("Event(1, hovered): %v", err)
	}
	if err := o.Event(7, "clicked", dbus.MakeVariant(""), 0); err == nil {
		t.Error("Event(7, clicked) on a missing entry: got no error")
	}
	idErrors, err := o.EventGroup([]menuEvent{
		{ID: 1, EventID: "clicked"},
		{ID: 8, EventID: "clicked"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(idErrors) != 1 || idErrors[0] != 8 {
		t.Errorf("EventGroup idErrors = %v, want [8]", idErrors)
	}
	if strings.Join(clicks, ",") != "quit,open" {
		t.Errorf("clicks = %v, want [quit open]", clicks)
	}

	props, _ := o.GetGroupProperties([]int32{0, 1, 42}, nil)
	if len(props) != 2 {
		t.Errorf("GetGroupProperties returned %d nodes, want 2", len(props))
	}
	if v, err := o.GetProperty(1, "label"); err != nil || v.Value() != "Open" {
		t.Errorf("GetProperty(1, label) = %v, %v, want Open", v, err)
	}
}

// fakeWatcher records the services registered with it.
type fakeWatcher struct {
	mu       sync.Mutex
	services []string
	got      chan struct{}
}

func (w *fakeWatcher) RegisterStatusNotifierItem(service string) *dbus.Error {
	w.mu.Lock()
	w.services = append(w.services, service)
	w.mu.Unlock()
	w.got <- struct{}{}
	return nil
}

// startBus runs a private session bus for the test and points
// DBUS_SESSION_BUS_ADDRESS at it.
func startBus(t *testing.T) {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon not installed")
	}
	cmd := exec.CommandContext(t.Context(), daemon, "--session", "--nofork", "--print-address=1")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Skipf("dbus-daemon did not start: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	addr, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatalf("read bus address: %v", err)
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", strings.TrimSpace(addr))
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestItemOnBus(t *testing.T) {
	startBus(t)

	host, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = host.Close() }()
	watcher := &fakeWatcher{got: make(chan struct{}, 4)}
	if err := host.Export(watcher, watcherPath, watcherName); err != nil {
		t.Fatal(err)
	}
	if _, err := host.RequestName(watcherName, dbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}

	activated := make(chan struct{}, 1)
	clicked := make(chan struct{}, 1)
	it, err := Start(Options{
		ID:         "test.tray",
		Title:      "Test Tray",
		IconName:   "test-symbolic",
		OnActivate: func() { activated <- struct{}{} },
		Menu: []MenuItem{
			{Label: "Open", OnClick: func() { clicked <- struct{}{} }},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = it.Close() }()

	waitFor(t, watcher.got, "the registration")
	watcher.mu.Lock()
	service := watcher.services[0]
	watcher.mu.Unlock()
	if service != it.busName {
		t.Errorf("registered %q, want %q", service, it.busName)
	}

	item := host.Object(service, itemPath)
	menuRef, err := item.GetProperty(itemIface + ".Menu")
	if err != nil || menuRef.Value() != menuPath {
		t.Errorf("Menu property = %v, %v, want %s", menuRef, err, menuPath)
	}
	iconName, err := item.GetProperty(itemIface + ".IconName")
	if err != nil || iconName.Value() != "test-symbolic" {
		t.Errorf("IconName property = %v, %v, want test-symbolic", iconName, err)
	}

	if err := item.Call(itemIface+".Activate", 0, int32(0), int32(0)).Err; err != nil {
		t.Fatalf("Activate: %v", err)
	}
	waitFor(t, activated, "the activation callback")

	menu := host.Object(service, menuPath)
	var revision uint32
	var root layout
	if err := menu.Call(menuIface+".GetLayout", 0, int32(0), int32(-1), []string{}).Store(&revision, &root); err != nil {
		t.Fatalf("GetLayout: %v", err)
	}
	if len(root.Children) != 1 {
		t.Fatalf("GetLayout: %d children, want 1", len(root.Children))
	}
	if err := menu.Call(menuIface+".Event", 0, int32(1), "clicked", dbus.MakeVariant(""), uint32(0)).Err; err != nil {
		t.Fatalf("Event: %v", err)
	}
	waitFor(t, clicked, "the menu click callback")

	if err := it.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	var hasOwner bool
	if err := host.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, service).Store(&hasOwner); err != nil {
		t.Fatal(err)
	}
	if hasOwner {
		t.Errorf("%s still owned after Close", service)
	}
}
