// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestPathConversion validates ToPath and ToString conversions
func TestPathConversion(t *testing.T) {
	s := "interfaces/interface[name=5]/state/id"
	p := ToPath(s)
	ps := ToString(p)
	assert.Equal(t, s, ps)
	assert.Len(t, p.Elem, 4)

	s = "/interfaces/interface[name=5]/state/id"
	p = ToPath(s)
	ps = ToString(p)
	assert.Equal(t, s[1:], ps)
	assert.Len(t, p.Elem, 4)
}

func TestSplitJoinPath(t *testing.T) {
	path := "interfaces/interface[name=5]/state/id"
	segments := SplitPath(path)
	path1 := JoinPath(segments)
	assert.Equal(t, path, path1)
}

func TestSubpath(t *testing.T) {
	path := "interfaces"
	path1 := Subpath(path, "interface", map[string]string{"name": "5"})
	assert.Equal(t, "interfaces/interface[name=5]", path1)

	path2 := Subpath("interfaces/interface[name=5]", "state", map[string]string{})
	assert.Equal(t, "interfaces/interface[name=5]/state", path2)
}

func TestNameKey(t *testing.T) {
	path, kv, w := NameKey("interface[name=5]")
	assert.Equal(t, "interface", path)
	assert.False(t, w)
	assert.Len(t, kv, 1)

	path, kv, w = NameKey("interface[name=...]")
	assert.Equal(t, "interface", path)
	assert.True(t, w)
	assert.Len(t, kv, 1)

}
