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

package config

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// mergeChanged compares before and after, both decoded from data, and
// returns data with only the top-level keys whose encoded value differs
// replaced by after's (or removed, when after omits them), together with
// the changed keys in after's field order followed by removed ones. Every
// other key, with its comments, is copied from data unchanged.
func mergeChanged(data []byte, before, after *Config) ([]byte, []string, error) {
	oldKeys, oldVals, err := encodeTop(before)
	if err != nil {
		return nil, nil, err
	}
	newKeys, newVals, err := encodeTop(after)
	if err != nil {
		return nil, nil, err
	}

	var changed []string
	for _, key := range newKeys {
		if !sameNode(oldVals[key], newVals[key]) {
			changed = append(changed, key)
		}
	}
	for _, key := range oldKeys {
		if _, ok := newVals[key]; !ok {
			changed = append(changed, key)
		}
	}
	if len(changed) == 0 {
		return data, nil, nil
	}

	doc, err := parseDocument(data)
	if err != nil {
		return nil, nil, err
	}
	root := doc.Content[0]
	for _, key := range changed {
		setKey(root, key, newVals[key])
	}
	out, err := encodeNode(doc)
	if err != nil {
		return nil, nil, err
	}
	return out, changed, nil
}

// encodeTop encodes cfg and returns its top-level keys in field order and
// each key's value node. Fields left out by omitempty are absent.
func encodeTop(cfg *Config) ([]string, map[string]*yaml.Node, error) {
	var n yaml.Node
	if err := n.Encode(cfg); err != nil {
		return nil, nil, fmt.Errorf("encode config: %w", err)
	}
	keys := make([]string, 0, len(n.Content)/2)
	vals := make(map[string]*yaml.Node, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i].Value
		keys = append(keys, key)
		vals[key] = n.Content[i+1]
	}
	return keys, vals, nil
}

// sameNode reports whether a and b encode to the same YAML; a nil node
// equals only another nil node.
func sameNode(a, b *yaml.Node) bool {
	if a == nil || b == nil {
		return a == b
	}
	ea, errA := encodeNode(a)
	eb, errB := encodeNode(b)
	return errA == nil && errB == nil && bytes.Equal(ea, eb)
}

// parseDocument parses data into a document node whose content is one
// mapping node; empty data (or data holding only comments) gives an
// empty mapping.
func parseDocument(data []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode}
	}
	if len(doc.Content) == 0 {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	if doc.Kind != yaml.DocumentNode || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parse config: the top level is not a mapping")
	}
	return &doc, nil
}

// setKey replaces the value of key in the mapping node m with val,
// appending the key when m doesn't have it; a nil val removes the key.
func setKey(m *yaml.Node, key string, val *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value != key {
			continue
		}
		if val == nil {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
		} else {
			m.Content[i+1] = val
		}
		return
	}
	if val != nil {
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, val)
	}
}

// encodeNode encodes n the way yaml.Marshal encodes a value (four-space
// indentation).
func encodeNode(n *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(4)
	if err := enc.Encode(n); err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	return buf.Bytes(), nil
}
