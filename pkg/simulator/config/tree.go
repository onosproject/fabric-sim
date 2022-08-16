// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package config contains the configuration tree
package config

import (
	"github.com/onosproject/fabric-sim/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
)

//var log = logging.GetLogger("config")

// Node represents a single node in the configuration tree
type Node struct {
	path     string
	name     string
	key      map[string]string
	value    *gnmi.TypedValue
	children map[string][]*Node
}

// Path returns the full node path
func (n *Node) Path() string {
	return n.path
}

// Name returns the node name
func (n *Node) Name() string {
	return n.name
}

// Key returns the node key
func (n *Node) Key() map[string]string {
	return n.key
}

// Value returns the node value; nil if node is a parent node to other children
func (n *Node) Value() *gnmi.TypedValue {
	return n.value
}

// NewRoot creates a new configuration tree root
func NewRoot() *Node {
	return &Node{
		path:     "",
		name:     "",
		key:      make(map[string]string),
		children: make(map[string][]*Node),
	}
}

// MatchesKey returns true if the node's key matches the given key, which can include wild-cards
func (n *Node) MatchesKey(key map[string]string) bool {
	if len(key) != len(n.key) {
		return false
	}

	for k, v := range key {
		if v != "..." && v != n.key[k] {
			return false
		}
	}

	return true
}

// Add adds a child node, identified by name and key; if the node already exists, it will
// not be replaced, but it's value will be updated
func (n *Node) Add(name string, key map[string]string, value *gnmi.TypedValue) *Node {
	node := n.Get(name, key)
	if node == nil {
		node = &Node{
			path:     utils.Subpath(n.path, name, key),
			name:     name,
			key:      key,
			value:    value,
			children: make(map[string][]*Node),
		}
		n.children[name] = append(n.children[name], node)
	}
	node.value = value
	return node
}

// Get finds the immediate child node, identified by name and key; returns nil if the node does not exist
func (n *Node) Get(name string, key map[string]string) *Node {
	children, ok := n.children[name]
	if !ok {
		return nil
	}

	// Iterate over the children with the same name and find the one with matching key
	for _, child := range children {
		if child.MatchesKey(key) {
			return child
		}
	}
	return nil
}

// Delete deletes the specified child node, identified by name and key;
// returns the deleted node or nil if node did not exist
func (n *Node) Delete(name string, key map[string]string) *Node {
	children, ok := n.children[name]
	if !ok {
		return nil
	}

	// Iterate over the children with the same name and find the one with matching key
	for i, child := range children {
		if child.MatchesKey(key) {
			// If key matches, delete the child and return it
			children[i] = children[len(children)-1]
			children[len(children)-1] = nil
			n.children[name] = children[:len(children)-1] // Truncate
			return child
		}
	}
	return nil
}

// AddPath iteratively adds the nodes along the given path and returns the last node added
func (n *Node) AddPath(path string, value *gnmi.TypedValue) *Node {
	current := n
	segments := utils.SplitPath(path)
	for i := 0; i < len(segments)-1; i++ {
		name, key, _ := utils.NameKey(segments[i])
		current = current.Add(name, key, nil)
	}
	name, key, _ := utils.NameKey(segments[len(segments)-1])
	return current.Add(name, key, value)
}

// GetPath iteratively gets the nodes along the given path and returns the last node specified; nil if not found
func (n *Node) GetPath(path string) *Node {
	current := n
	segments := utils.SplitPath(path)
	for i := 0; i < len(segments)-1; i++ {
		name, key, _ := utils.NameKey(segments[i])
		if current = current.Get(name, key); current == nil {
			return nil
		}
	}
	name, key, _ := utils.NameKey(segments[len(segments)-1])
	return current.Get(name, key)
}

// ReplacePath performs like AddPath, but replaces the leaf node entirely, including its child nodes
func (n *Node) ReplacePath(path string, value *gnmi.TypedValue) *Node {
	leafNode := n.AddPath(path, value)
	leafNode.children = make(map[string][]*Node)
	// TODO: properly implement handling default values, etc. if needed
	return leafNode
}

// DeletePath iteratively gets the nodes along the given path and deletes the last node specified; nil if not found
func (n *Node) DeletePath(path string) *Node {
	current := n
	segments := utils.SplitPath(path)
	for i := 0; i < len(segments)-1; i++ {
		name, key, _ := utils.NameKey(segments[i])
		if current = current.Get(name, key); current == nil {
			return nil
		}
	}
	name, key, _ := utils.NameKey(segments[len(segments)-1])
	return current.Delete(name, key)
}

// FindAll finds all nodes matching the specified path, which can include wildcard "..." as key value
func (n *Node) FindAll(path string) []*Node {
	nodes := make([]*Node, 0)

	current := n
	segments := utils.SplitPath(path)
	for i := 0; i < len(segments)-1; i++ {
		name, key, hasWildcard := utils.NameKey(segments[i])
		if hasWildcard {
			if children, ok := current.children[name]; ok {
				for _, child := range children {
					if child.MatchesKey(key) {
						nodes = append(nodes, child.FindAll(utils.JoinPath(segments[i+1:]))...)
					}
				}
			}
			return nodes
		}
		if current = current.Get(name, key); current == nil {
			return nil
		}
	}
	name, key, _ := utils.NameKey(segments[len(segments)-1])
	if current = current.Get(name, key); current != nil {
		nodes = append(nodes, current.GatherAllDescendants()...)
	}

	return nodes
}

// GatherAllDescendants gathers all descendants of this node
func (n *Node) GatherAllDescendants() []*Node {
	if n.value != nil {
		return []*Node{n}
	}

	nodes := make([]*Node, 0)
	for _, children := range n.children {
		for _, child := range children {
			nodes = append(nodes, child.GatherAllDescendants()...)
		}
	}
	return nodes
}
