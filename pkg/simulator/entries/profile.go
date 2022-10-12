// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package entries

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
)

// ActionProfileMember represents a P4 action profile member
type ActionProfileMember struct {
	entry *p4api.ActionProfileMember
}

// ActionProfileGroup represents a P4 action profile group
type ActionProfileGroup struct {
	entry *p4api.ActionProfileGroup
}

// ActionProfile represents a P4 action profile instance
type ActionProfile struct {
	info    *p4info.ActionProfile
	members map[uint32]*ActionProfileMember
	groups  map[uint32]*ActionProfileGroup
}

// ActionProfiles represents a set of P4 action profiles
type ActionProfiles struct {
	profiles map[uint32]*ActionProfile
}

// NewActionProfiles creates a new action profiles
func NewActionProfiles(actionProfilesInfo []*p4info.ActionProfile) *ActionProfiles {
	gs := &ActionProfiles{
		profiles: make(map[uint32]*ActionProfile, len(actionProfilesInfo)),
	}
	for _, pi := range actionProfilesInfo {
		gs.profiles[pi.Preamble.Id] = gs.NewActionProfile(pi)
	}
	return gs
}

// NewActionProfile creates a new action profile
func (aps *ActionProfiles) NewActionProfile(info *p4info.ActionProfile) *ActionProfile {
	return &ActionProfile{
		info:    info,
		members: make(map[uint32]*ActionProfileMember),
		groups:  make(map[uint32]*ActionProfileGroup),
	}
}

// ModifyActionProfileMember modifies the specified action profile member
func (aps *ActionProfiles) ModifyActionProfileMember(entry *p4api.ActionProfileMember, insert bool) error {
	profile, ok := aps.profiles[entry.ActionProfileId]
	if !ok {
		return errors.NewNotFound("action profile not found")
	}
	return profile.ModifyActionProfileMember(entry, insert)
}

// ReadActionProfileMembers reads action profile members
func (aps *ActionProfiles) ReadActionProfileMembers(entry *p4api.ActionProfileMember, sender BatchSender) error {
	profile, ok := aps.profiles[entry.ActionProfileId]
	if !ok {
		return errors.NewNotFound("action profile not found")
	}
	return profile.ReadActionProfileMembers(sender)
}

// DeleteActionProfileMember deletes the specified member entry from its action profile
func (aps *ActionProfiles) DeleteActionProfileMember(entry *p4api.ActionProfileMember) error {
	profile, ok := aps.profiles[entry.ActionProfileId]
	if !ok {
		return errors.NewNotFound("action profile not found")
	}
	return profile.DeleteActionProfileMember(entry)
}

// ModifyActionProfileGroup modifies the specified action profile group
func (aps *ActionProfiles) ModifyActionProfileGroup(entry *p4api.ActionProfileGroup, insert bool) error {
	profile, ok := aps.profiles[entry.ActionProfileId]
	if !ok {
		return errors.NewNotFound("action profile not found")
	}
	return profile.ModifyActionProfileGroup(entry, insert)
}

// ReadActionProfileGroups reads action profile groups
func (aps *ActionProfiles) ReadActionProfileGroups(entry *p4api.ActionProfileGroup, sender BatchSender) error {
	profile, ok := aps.profiles[entry.ActionProfileId]
	if !ok {
		return errors.NewNotFound("action profile not found")
	}
	return profile.ReadActionProfileGroups(sender)
}

// DeleteActionProfileGroup deletes the specified action profile group
func (aps *ActionProfiles) DeleteActionProfileGroup(entry *p4api.ActionProfileGroup) error {
	profile, ok := aps.profiles[entry.ActionProfileId]
	if !ok {
		return errors.NewNotFound("action profile not found")
	}
	return profile.DeleteActionProfileGroup(entry)
}

// ModifyActionProfileMember modifies the specified member entry
func (ap ActionProfile) ModifyActionProfileMember(entry *p4api.ActionProfileMember, insert bool) error {
	member, ok := ap.members[entry.MemberId]

	// If the entry exists, and we're supposed to do a new insert, raise error
	if ok && insert {
		return errors.NewAlreadyExists("entry already exists: %v", entry)
	}

	// If the entry doesn't exist, and we're supposed to modify, raise error
	if !ok && !insert {
		return errors.NewNotFound("entry doesn't exist: %v", entry)
	}

	// If the entry doesn't exist and we're supposed to do insert, well... do it
	if !ok && insert {
		if int64(len(ap.members)) > ap.info.Size {
			return errors.NewUnavailable("resource exhausted: %v", entry)
		}
		member = &ActionProfileMember{}
		ap.members[entry.MemberId] = member
	}

	// Otherwise, update the entry
	member.entry = entry
	return nil
}

// DeleteActionProfileMember deletes the specified member entry
func (ap ActionProfile) DeleteActionProfileMember(entry *p4api.ActionProfileMember) error {
	delete(ap.members, entry.MemberId)
	return nil
}

// ReadActionProfileMembers sends all members of the profile to the specified sender
func (ap ActionProfile) ReadActionProfileMembers(sender BatchSender) error {
	buffer := newBuffer(sender)
	for _, member := range ap.members {
		if err := buffer.sendEntity(&p4api.Entity{Entity: &p4api.Entity_ActionProfileMember{ActionProfileMember: member.entry}}); err != nil {
			return err
		}
	}
	return buffer.flush()
}

// ModifyActionProfileGroup modifies the specified group in this action profile
func (ap ActionProfile) ModifyActionProfileGroup(entry *p4api.ActionProfileGroup, insert bool) error {
	group, ok := ap.groups[entry.GroupId]

	// If the entry exists, and we're supposed to do a new insert, raise error
	if ok && insert {
		return errors.NewAlreadyExists("entry already exists: %v", entry)
	}

	// If the entry doesn't exist, and we're supposed to modify, raise error
	if !ok && !insert {
		return errors.NewNotFound("entry doesn't exist: %v", entry)
	}

	// If the entry doesn't exist and we're supposed to do insert, well... do it
	if !ok && insert {
		if int64(len(ap.groups)) > ap.info.Size {
			return errors.NewUnavailable("resource exhausted: %v", entry)
		}
		group = &ActionProfileGroup{}
		ap.groups[entry.GroupId] = group
	}

	// Otherwise, update the entry
	group.entry = entry
	return nil

}

// ReadActionProfileGroups sends all groups of the profile to the specified sender
func (ap ActionProfile) ReadActionProfileGroups(sender BatchSender) error {
	buffer := newBuffer(sender)
	for _, group := range ap.groups {
		if err := buffer.sendEntity(&p4api.Entity{Entity: &p4api.Entity_ActionProfileGroup{ActionProfileGroup: group.entry}}); err != nil {
			return err
		}
	}
	return buffer.flush()
}

// DeleteActionProfileGroup deletes the specified group from this action profile
func (ap ActionProfile) DeleteActionProfileGroup(entry *p4api.ActionProfileGroup) error {
	delete(ap.groups, entry.GroupId)
	return nil
}
