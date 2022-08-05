// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package utils contains various utilities for working with P4Info and P4RT entities
package utils

import (
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	"google.golang.org/protobuf/encoding/prototext"
	"io/ioutil"
)

// LoadP4Info loads the specified file containing protoJSON representation of a P4Info and returns its descriptor
func LoadP4Info(path string) (*p4info.P4Info, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info := &p4info.P4Info{}
	err = prototext.Unmarshal(data, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}
