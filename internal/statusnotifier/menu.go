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
	"sync"

	"github.com/godbus/dbus/v5"
)

// rootID is the dbusmenu ID of the invisible root holding the entries;
// entry i of the menu has ID i+1.
const rootID int32 = 0

// layout is the D-Bus representation (ia{sv}av) of a dbusmenu node.
type layout struct {
	ID       int32
	Props    map[string]dbus.Variant
	Children []dbus.Variant
}

// itemProperties is the D-Bus representation (ia{sv}) of one node's
// properties.
type itemProperties struct {
	ID    int32
	Props map[string]dbus.Variant
}

// menuEvent is the D-Bus representation (isvu) of one EventGroup entry.
type menuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

// menuModel holds the menu entries and their layout revision; mu guards
// both.
type menuModel struct {
	mu       sync.Mutex
	items    []MenuItem
	revision uint32
}

func newMenuModel(items []MenuItem) *menuModel {
	return &menuModel{items: items, revision: 1}
}

// set replaces the entries and returns the new revision.
func (m *menuModel) set(items []MenuItem) uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = items
	m.revision++
	return m.revision
}

// props returns the dbusmenu properties of the node id, and whether it
// exists. The caller holds mu.
func (m *menuModel) props(id int32) (map[string]dbus.Variant, bool) {
	if id == rootID {
		return map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")}, true
	}
	if id < 1 || int(id) > len(m.items) {
		return nil, false
	}
	item := m.items[id-1]
	if item.Separator {
		return map[string]dbus.Variant{
			"type":    dbus.MakeVariant("separator"),
			"visible": dbus.MakeVariant(true),
		}, true
	}
	return map[string]dbus.Variant{
		"label":   dbus.MakeVariant(item.Label),
		"enabled": dbus.MakeVariant(true),
		"visible": dbus.MakeVariant(true),
	}, true
}

// layout returns the revision and the subtree under parent: the root with
// every entry, or a single entry without children.
func (m *menuModel) layout(parent int32) (uint32, layout, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	props, ok := m.props(parent)
	if !ok {
		return m.revision, layout{}, false
	}
	node := layout{ID: parent, Props: props, Children: []dbus.Variant{}}
	if parent == rootID {
		for i := range m.items {
			id := int32(i + 1)
			childProps, _ := m.props(id)
			node.Children = append(node.Children, dbus.MakeVariant(layout{
				ID: id, Props: childProps, Children: []dbus.Variant{},
			}))
		}
	}
	return m.revision, node, true
}

// groupProperties returns the properties of each existing node in ids.
func (m *menuModel) groupProperties(ids []int32) []itemProperties {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []itemProperties{}
	for _, id := range ids {
		if props, ok := m.props(id); ok {
			out = append(out, itemProperties{ID: id, Props: props})
		}
	}
	return out
}

// property returns one property of the node id.
func (m *menuModel) property(id int32, name string) (dbus.Variant, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	props, ok := m.props(id)
	if !ok {
		return dbus.Variant{}, false
	}
	v, ok := props[name]
	return v, ok
}

// click runs the OnClick of the entry id, outside mu, and reports whether
// the entry exists.
func (m *menuModel) click(id int32) bool {
	m.mu.Lock()
	var onClick func()
	ok := id >= 1 && int(id) <= len(m.items)
	if ok {
		onClick = m.items[id-1].OnClick
	}
	m.mu.Unlock()
	if onClick != nil {
		onClick()
	}
	return ok
}

// unknownItem is the D-Bus error for an ID the menu doesn't have.
func unknownItem() *dbus.Error {
	return dbus.NewError("com.canonical.dbusmenu.Error.UnknownId", []any{"unknown menu item"})
}

// menuObject implements the com.canonical.dbusmenu methods.
type menuObject struct {
	model *menuModel
}

// GetLayout returns the menu revision and the subtree under parentID.
func (o *menuObject) GetLayout(parentID, _ int32, _ []string) (uint32, layout, *dbus.Error) {
	revision, node, ok := o.model.layout(parentID)
	if !ok {
		return 0, layout{}, unknownItem()
	}
	return revision, node, nil
}

// GetGroupProperties returns the properties of the nodes in ids.
func (o *menuObject) GetGroupProperties(ids []int32, _ []string) ([]itemProperties, *dbus.Error) {
	return o.model.groupProperties(ids), nil
}

// GetProperty returns one property of the node id.
func (o *menuObject) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	v, ok := o.model.property(id, name)
	if !ok {
		return dbus.Variant{}, unknownItem()
	}
	return v, nil
}

// Event runs the entry's OnClick on a "clicked" event; other events are
// ignored.
func (o *menuObject) Event(id int32, eventID string, _ dbus.Variant, _ uint32) *dbus.Error {
	if eventID == "clicked" && !o.model.click(id) {
		return unknownItem()
	}
	return nil
}

// EventGroup handles each event like Event and returns the IDs not found.
func (o *menuObject) EventGroup(events []menuEvent) ([]int32, *dbus.Error) {
	idErrors := []int32{}
	for _, e := range events {
		if e.EventID == "clicked" && !o.model.click(e.ID) {
			idErrors = append(idErrors, e.ID)
		}
	}
	return idErrors, nil
}

// AboutToShow reports that the layout needs no update.
func (o *menuObject) AboutToShow(_ int32) (bool, *dbus.Error) { return false, nil }

// AboutToShowGroup reports that no layout needs an update.
func (o *menuObject) AboutToShowGroup(_ []int32) ([]int32, []int32, *dbus.Error) {
	return []int32{}, []int32{}, nil
}
