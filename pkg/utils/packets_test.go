// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestPathConversion validates ToPath and ToString conversions
func TestIP(t *testing.T) {
	assert.Len(t, IP("1.2.3.4"), 4)
	assert.Equal(t, IP("1.2.3.4"), []byte{0x1, 0x2, 0x3, 0x4})
	assert.Len(t, MAC("11:22:33:44:55:66"), 6)
	assert.Equal(t, MAC("11:22:33:44:55:66"), []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66})
}
