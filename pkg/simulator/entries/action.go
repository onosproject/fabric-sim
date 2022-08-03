// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package entries

import (
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
)

// Action represents a P4 action
type Action struct {
	Action *p4info.Action
}

// NewAction creates a new action
func NewAction(table *p4info.Action) *Action {
	return &Action{
		Action: table,
	}
}
