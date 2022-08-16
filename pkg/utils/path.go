// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"github.com/openconfig/gnmi/proto/gnmi"
	"sort"
	"strings"
)

// Separates segments of path in its string representation
const pathSeparator = "/"

// SplitPath splits the given string path into its segments
func SplitPath(path string) []string {
	return strings.Split(strings.TrimPrefix(path, pathSeparator), pathSeparator)
}

// JoinPath joins the given path segments into a path string
func JoinPath(segments []string) string {
	return strings.Join(segments, pathSeparator)
}

// Subpath creates a subpath of a given path
func Subpath(path string, name string, key map[string]string) string {
	if len(key) == 0 {
		return fmt.Sprintf("%s/%s", path, name)
	}
	return strings.TrimPrefix(fmt.Sprintf("%s/%s[%s]", path, name, keyString(key)), pathSeparator)
}

// ToPath produces a gNMI path from the given string representation
func ToPath(path string) *gnmi.Path {
	segments := strings.Split(strings.TrimPrefix(path, pathSeparator), pathSeparator)
	elements := make([]*gnmi.PathElem, 0, len(segments))
	for _, segment := range segments {
		name, key, _ := NameKey(segment)
		elements = append(elements, &gnmi.PathElem{
			Name: name,
			Key:  key,
		})
	}
	return &gnmi.Path{
		Elem: elements,
	}
}

// ToString produces a deterministic string representation of the given gNMI path structure
func ToString(path *gnmi.Path) string {
	segments := make([]string, 0, len(path.Elem))
	for _, pe := range path.Elem {
		if len(pe.Name) > 0 {
			segment := pe.Name
			if len(pe.Key) > 0 {
				segment = fmt.Sprintf("%s[%s]", pe.Name, keyString(pe.Key))
			}
			segments = append(segments, segment)
		}
	}
	return strings.Join(segments, pathSeparator)
}

// Return deterministic string representation of the given key map
func keyString(key map[string]string) string {
	sk := make([]string, 0, len(key))
	for k := range key {
		sk = append(sk, k)
	}
	sort.Strings(sk)

	pairs := make([]string, 0, len(key))
	for _, k := range sk {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, key[k]))
	}
	return strings.Join(pairs, ",")
}

// NameKey splits the string representation of the path segment into name and an optional key
func NameKey(e string) (string, map[string]string, bool) {
	hasWildcard := false
	fields := strings.Split(e, "[")
	if len(fields) == 1 {
		return fields[0], nil, false
	}
	name := fields[0]
	fields = strings.Split(strings.Split(fields[1], "]")[0], ",")
	key := make(map[string]string, len(fields))
	for _, field := range fields {
		kv := strings.Split(strings.TrimSpace(field), "=")
		if len(kv) == 2 {
			if kv[1] == "..." {
				hasWildcard = true
			}
			key[kv[0]] = kv[1]
		}
	}
	return name, key, hasWildcard
}
