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
