// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package entries

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
)

// PacketReplication represents packet replication engine constructs
type PacketReplication struct {
	multicasts    map[uint32]*p4api.MulticastGroupEntry
	cloneSessions map[uint32]*p4api.CloneSessionEntry
}

// NewPacketReplication creates store for P4 PRE constructs
func NewPacketReplication() *PacketReplication {
	return &PacketReplication{
		multicasts:    make(map[uint32]*p4api.MulticastGroupEntry),
		cloneSessions: make(map[uint32]*p4api.CloneSessionEntry),
	}
}

// ModifyMulticastGroupEntry modifies the specified multicast group entry
func (pr *PacketReplication) ModifyMulticastGroupEntry(entry *p4api.MulticastGroupEntry, insert bool) error {
	_, ok := pr.multicasts[entry.MulticastGroupId]

	// If the entry exists, and we're supposed to do a new insert, raise error
	if ok && insert {
		return errors.NewAlreadyExists("entry already exists: %v", entry)
	}

	// If the entry doesn't exist, and we're supposed to modify, raise error
	if !ok && !insert {
		return errors.NewNotFound("entry doesn't exist: %v", entry)
	}

	pr.multicasts[entry.MulticastGroupId] = entry
	return nil
}

// ReadMulticastGroupEntries sends all multicast group entries to the given sender
func (pr *PacketReplication) ReadMulticastGroupEntries(entry *p4api.MulticastGroupEntry, sender BatchSender) error {
	buffer := newBuffer(sender)
	for _, mge := range pr.multicasts {
		if err := buffer.sendEntity(&p4api.Entity{Entity: &p4api.Entity_PacketReplicationEngineEntry{
			PacketReplicationEngineEntry: &p4api.PacketReplicationEngineEntry{
				Type: &p4api.PacketReplicationEngineEntry_MulticastGroupEntry{MulticastGroupEntry: mge},
			}}}); err != nil {
			return err
		}
	}
	return buffer.flush()
}

// DeleteMulticastGroupEntry deletes the specified multicast group entry
func (pr *PacketReplication) DeleteMulticastGroupEntry(entry *p4api.MulticastGroupEntry) error {
	delete(pr.multicasts, entry.MulticastGroupId)
	return nil
}

// ModifyCloneSessionEntry modifies the specified clone session entry
func (pr *PacketReplication) ModifyCloneSessionEntry(entry *p4api.CloneSessionEntry, insert bool) error {
	_, ok := pr.cloneSessions[entry.SessionId]

	// If the entry exists, and we're supposed to do a new insert, raise error
	if ok && insert {
		return errors.NewAlreadyExists("entry already exists: %v", entry)
	}

	// If the entry doesn't exist, and we're supposed to modify, raise error
	if !ok && !insert {
		return errors.NewNotFound("entry doesn't exist: %v", entry)
	}

	pr.cloneSessions[entry.SessionId] = entry
	return nil
}

// ReadCloneSessionEntries sends all clone session entries to the given sender
func (pr *PacketReplication) ReadCloneSessionEntries(entry *p4api.CloneSessionEntry, sender BatchSender) error {
	buffer := newBuffer(sender)
	for _, cs := range pr.cloneSessions {
		if err := buffer.sendEntity(&p4api.Entity{Entity: &p4api.Entity_PacketReplicationEngineEntry{
			PacketReplicationEngineEntry: &p4api.PacketReplicationEngineEntry{
				Type: &p4api.PacketReplicationEngineEntry_CloneSessionEntry{CloneSessionEntry: cs},
			}}}); err != nil {
			return err
		}
	}
	return buffer.flush()
}

// DeleteCloneSessionEntry deletes the specified close session entry
func (pr *PacketReplication) DeleteCloneSessionEntry(entry *p4api.CloneSessionEntry) error {
	delete(pr.cloneSessions, entry.SessionId)
	return nil
}
