package ipset

import (
	"fmt"

	"github.com/pkg/errors"
)

type Member struct {
	Elem      string `xml:" elem" json:"elem"`
	Comment   string `xml:" comment,omitempty" json:"comment,omitempty"`
	Timeout   int    `xml:" timeout,omitempty" json:"timeout,omitempty"`
	Packets   int    `xml:" packets,omitempty" json:"packets,omitempty"`
	Bytes     int    `xml:" bytes,omitempty" json:"bytes,omitempty"`
	SBKMark   string `xml:" skbmark,omitempty" json:"skbmark,omitempty"`
	SKBPrio   string `xml:" skbprio,omitempty" json:"skbprio,omitempty"`
	SKBQueue  string `xml:" skbqueue,omitempty" json:"skbqueue,omitempty"`
	parentSet *Set
}

// NewMember creates a new set member, it takes an elemnt text
// a pointer to the set and a list of options that could be used to set various fields.
// If pointer to set type is provided function will return error when
// requested set type is incompatible with collection of options.
func NewMember(elem string, set *Set, opts ...MemberOpt) (*Member, error) {
	m := Member{parentSet: set, Elem: elem}

	for _, opt := range opts {
		err := opt(&m)
		if err != nil {
			return nil, err
		}
	}

	return &m, nil
}

type MemberOpt func(*Member) error

func OptComment(comment string) MemberOpt {
	return func(m *Member) error {
		if m.parentSet != nil && m.parentSet.Header != nil && m.parentSet.Header.Comment == nil {
			return errors.New("comment options used with incompatible set")
		}
		m.Comment = comment
		return nil
	}
}

func OptTimeout(timeout int) MemberOpt {
	return func(m *Member) error {
		if m.parentSet != nil && m.parentSet.Header != nil && m.parentSet.Header.Timeout == 0 {
			return errors.New("timeout options used with incompatible set")
		}
		m.Timeout = timeout
		return nil
	}
}

func OptPackets(packets int) MemberOpt {
	return func(m *Member) error {
		if m.parentSet != nil && m.parentSet.Header != nil && m.parentSet.Header.Counters == nil {
			return errors.New("packets options used with incompatible set")
		}
		m.Packets = packets
		return nil
	}
}

func OptBytes(bytes int) MemberOpt {
	return func(m *Member) error {
		if m.parentSet != nil && m.parentSet.Header != nil && m.parentSet.Header.Counters == nil {
			return errors.New("bytes options used with incompatible set")
		}
		m.Bytes = bytes
		return nil
	}
}

func OptSBKMark(sbkmark string) MemberOpt {
	return func(m *Member) error {
		if m.parentSet != nil && m.parentSet.Header != nil && m.parentSet.Header.SKBInfo == nil {
			return errors.New("skbmark options used with incompatible set")
		}
		m.SBKMark = sbkmark
		return nil
	}
}

func OptSKBPrio(skbprio string) MemberOpt {
	return func(m *Member) error {
		if m.parentSet != nil && m.parentSet.Header != nil && m.parentSet.Header.SKBInfo == nil {
			return errors.New("skbprio options used with incompatible set")
		}
		m.SKBPrio = skbprio
		return nil
	}
}

func (m Member) Render() string {
	var result string

	result += m.Elem

	if m.Comment != "" {
		result += fmt.Sprintf(" %s", m.Comment)
	}
	if m.Timeout != 0 {
		result += fmt.Sprintf(" %d", m.Timeout)
	}
	if m.Packets != 0 {
		result += fmt.Sprintf(" %d", m.Packets)
	}
	if m.Bytes != 0 {
		result += fmt.Sprintf(" %d", m.Bytes)
	}
	if m.SBKMark != "" {
		result += fmt.Sprintf(" %s", m.SBKMark)
	}
	if m.SKBPrio != "" {
		result += fmt.Sprintf(" %s", m.SKBPrio)
	}
	if m.SKBQueue != "" {
		result += fmt.Sprintf(" %s", m.SKBQueue)
	}

	return result
}
