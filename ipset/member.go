package ipset

import (
	"fmt"

	"github.com/pkg/errors"
)

type Member struct {
	Elem      string  `xml:" elem" json:"elem"`
	Comment   string  `xml:" comment,omitempty" json:"comment,omitempty"`
	NoMatch   *string `xml:" nomatch,omitempty" json:"nomatch,omitempty"`
	Timeout   int     `xml:" timeout,omitempty" json:"timeout,omitempty"`
	Packets   int     `xml:" packets,omitempty" json:"packets,omitempty"`
	Bytes     int     `xml:" bytes,omitempty" json:"bytes,omitempty"`
	SKBMark   string  `xml:" skbmark,omitempty" json:"skbmark,omitempty"`
	SKBPrio   string  `xml:" skbprio,omitempty" json:"skbprio,omitempty"`
	SKBQueue  string  `xml:" skbqueue,omitempty" json:"skbqueue,omitempty"`
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

func OptMemComment(comment string) MemberOpt {
	return func(m *Member) error {
		tm := &Member{Comment: comment}
		if err := validateMemberForSet(m.parentSet, tm); err != nil {
			return errors.New("comment options used with incompatible set")
		}
		m.Comment = comment
		return nil
	}
}

func OptMemTimeout(timeout int) MemberOpt {
	return func(m *Member) error {
		tm := &Member{Timeout: timeout}
		if err := validateMemberForSet(m.parentSet, tm); err != nil {
			return errors.New("timeout options used with incompatible set")
		}
		m.Timeout = timeout
		return nil
	}
}

func OptMemNomatch(m *Member) error {
	m.NoMatch = new(string)
	return nil
}

func OptMemPackets(packets int) MemberOpt {
	return func(m *Member) error {
		tm := &Member{Packets: packets}
		if err := validateMemberForSet(m.parentSet, tm); err != nil {
			return errors.New("packets options used with incompatible set")
		}
		m.Packets = packets
		return nil
	}
}

func OptMemBytes(bytes int) MemberOpt {
	return func(m *Member) error {
		tm := &Member{Bytes: bytes}
		if err := validateMemberForSet(m.parentSet, tm); err != nil {
			return errors.New("bytes options used with incompatible set")
		}
		m.Bytes = bytes
		return nil
	}
}

func OptMemSKBMark(skbmark string) MemberOpt {
	return func(m *Member) error {
		tm := &Member{SKBMark: skbmark}
		if err := validateMemberForSet(m.parentSet, tm); err != nil {
			return errors.New("skbmark options used with incompatible set")
		}
		m.SKBMark = skbmark
		return nil
	}
}

func OptMemSKBPrio(skbprio string) MemberOpt {
	return func(m *Member) error {
		tm := &Member{SKBPrio: skbprio}
		if err := validateMemberForSet(m.parentSet, tm); err != nil {
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
		result += fmt.Sprintf(" comment %s", m.Comment)
	}
	if m.NoMatch != nil {
		result += fmt.Sprintf(" nomatch")
	}
	if m.Timeout != 0 {
		result += fmt.Sprintf(" timeout %d", m.Timeout)
	}
	if m.Packets != 0 {
		result += fmt.Sprintf(" packets %d", m.Packets)
	}
	if m.Bytes != 0 {
		result += fmt.Sprintf(" bytes %d", m.Bytes)
	}
	if m.SKBMark != "" {
		result += fmt.Sprintf(" skbmark %s", m.SKBMark)
	}
	if m.SKBPrio != "" {
		result += fmt.Sprintf(" skbprio %s", m.SKBPrio)
	}
	if m.SKBQueue != "" {
		result += fmt.Sprintf(" skbqueue %s", m.SKBQueue)
	}

	return result
}

const (
	MemberFamilyInet  = "inet"
	MemberFamilyInet6 = "inet6"
)
