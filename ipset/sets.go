package ipset

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// Set represents ipset set.
type Set struct {
	Name   string `xml:" name,attr"  json:",omitempty"`
	Header Header `xml:" header,omitempty" json:"header,omitempty"`
	//Header   *Header   `xml:" header,omitempty" json:"header,omitempty"`
	Members  []Member `xml:" members>member,omitempty" json:"members,omitempty"`
	Revision int      `xml:" revision,omitempty" json:"revision,omitempty"`
	Type     SetType  `xml:" type,omitempty" json:"type,omitempty"`
}

func NewSet(name string, sType SetType, options ...SetOpt) (*Set, error) {
	s := Set{Name: name, Type: sType}

	for _, opt := range options {
		err := opt(&s)
		if err != nil {
			return nil, err
		}
	}

	return &s, nil
}

// Render return string reprsentation of the set compatible with `ipset restore`
// or with interactive session.
func (s *Set) Render(rType RenderType) string {
	var result string

	if s == nil {
		return result
	}

	switch rType {
	// only create set
	case RenderCreate:
		result += fmt.Sprintf("create %s %s %s\n", s.Name, s.Type, s.Header.render())
	// normal save, render everything as create/add
	case RenderSave:
		result += fmt.Sprintf("create %s %s %s\n", s.Name, s.Type, s.Header.render())
		for _, member := range s.Members {
			result += fmt.Sprintf("add %s %s\n", s.Name, member.Render())
		}
	// only add members
	case RenderAdd:
		for _, member := range s.Members {
			result += fmt.Sprintf("add %s %s\n", s.Name, member.Render())
		}
	// only delete members
	case RenderDelete:
		for _, member := range s.Members {
			result += fmt.Sprintf("del %s %s\n", s.Name, member.Render())
		}
	// only set name for test
	case RenderTest:
		result += fmt.Sprintf("test %s", s.Name)
		if len(s.Members) == 1 {
			result += fmt.Sprintf(" %s", s.Members[0].Elem)
		}
	// only flush set
	case RenderFlush:
		result += fmt.Sprintf("flush %s\n", s.Name)
	// only destroy set
	case RenderDestroy:
		result += fmt.Sprintf("destroy %s\n", s.Name)
	}

	return result
}

func (s *Set) AddMember(m *Member) error {
	err := validateMemberForSet(s, m)
	if err != nil {
		return err
	}
	s.Members = append(s.Members, *m)

	return nil
}

type SetType string

const (
	SetBitmapIp       = "bitmap:ip"
	SetBitmapIpMac    = "bitmap:ip,mac"
	SetBitmapPort     = "bitmap:port"
	SetHashIp         = "hash:ip"
	SetHashMac        = "hash:mac"
	SetHashNet        = "hash:net"
	SetHashNetNet     = "hash:net,net"
	SetHashIpPort     = "hash:ip,port"
	SetHashNetPort    = "hash:net,port"
	SetHashIpPortIp   = "hash:ip,port,ip"
	SetHashIpPortNet  = "hash:ip,port,net"
	SetHashIpMark     = "hash:ip,mark"
	SetHashNetPortNet = "hash:net,port,net"
	SetHashNetIface   = "hash:net,iface"
	SetListSet        = "list:set"
)

type SetOpt func(*Set) error

func OptSetRevision(revision int) SetOpt {
	return func(s *Set) error {
		s.Revision = revision
		return nil
	}
}

func OptSetSize(size int) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{Size: size}); err != nil {
			return err
		}
		s.Header.Size = size
		return nil
	}
}

func OptSetFamily(family string) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{Family: family}); err != nil {
			return err
		}
		s.Header.Family = family
		return nil
	}
}

func OptSetRange(srange string) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{Range: srange}); err != nil {
			return err
		}
		s.Header.Range = srange
		return nil
	}
}

func OptSetHashsize(hashsize int) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{Hashsize: hashsize}); err != nil {
			return err
		}
		s.Header.Hashsize = hashsize
		return nil
	}
}

func OptSetMaxelem(maxelem int) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{Maxelem: maxelem}); err != nil {
			return err
		}
		s.Header.Maxelem = maxelem
		return nil
	}
}

func OptSetReferences(references int) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{References: references}); err != nil {
			return err
		}
		s.Header.References = references
		return nil
	}
}

func OptSetTimeout(timeout int) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{Timeout: timeout}); err != nil {
			return err
		}
		s.Header.Timeout = timeout
		return nil
	}
}

func OptSetNetmask(netmask int) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{Netmask: netmask}); err != nil {
			return err
		}
		s.Header.Netmask = netmask
		return nil
	}
}

func OptSetCounters(counters string) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{Counters: &counters}); err != nil {
			return err
		}
		s.Header.Counters = &counters
		return nil
	}
}

func OptSetComment(comment string) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{Comment: &comment}); err != nil {
			return err
		}
		s.Header.Comment = &comment
		return nil
	}
}

func OptSetSKBInfo(skbinfo string) SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{SKBInfo: &skbinfo}); err != nil {
			return err
		}
		s.Header.SKBInfo = &skbinfo
		return nil
	}
}

func OptSetForceadd() SetOpt {
	return func(s *Set) error {
		if err := validateSetHeader(s.Type, Header{Forceadd: NoVal}); err != nil {
			return err
		}
		s.Header.Forceadd = NoVal
		return nil
	}
}

func validateMemberForSet(s *Set, m *Member) error {
	if s == nil || m == nil {
		//	if s == nil || m == nil || s.Header == nil {
		return nil
	}

	if s.Header.Comment == nil && m.Comment != "" {
		return errors.New("comment options used with incompatible set")
	}

	if s.Header.Timeout == 0 && m.Timeout != 0 {
		return errors.New("timeout options used with incompatible set")
	}

	if s.Header.Counters == nil && m.Packets != 0 {
		return errors.New("packets options used with incompatible set")
	}

	if s.Header.Counters == nil && m.Bytes != 0 {
		return errors.New("bytes options used with incompatible set")
	}

	if s.Header.SKBInfo == nil && m.SKBMark != "" {
		return errors.New("skbmark options used with incompatible set")
	}

	if s.Header.SKBInfo == nil && m.SKBPrio != "" {
		return errors.New("skbprio options used with incompatible set")
	}

	if s.Header.SKBInfo == nil && m.SKBQueue != "" {
		return errors.New("skbqueue options used with incompatible set")
	}

	return nil
}

func validateSetHeader(sType SetType, header Header) error {
	/*
		{ "Family","family", `!= ""` },
		{ "Range","range", `!= ""` },
		{ "Hashsize","hashsize", "!= 0" },
		{ "Maxelem","maxelem", "!= 0" },
		{ "Memsize","memsize", "!= 0" },
		{ "References","references", "!= 0" },
		{ "Timeout","timeout", "!= 0" },
		{ "Netmask","netmask", "!= 0" },
		{ "Size","size", "!= 0" },
		{ "Counters","counters", "!= nil" },
		{ "Comment","comment", "!= nil" },
		{ "SKBInfo","skbinfo", "!= nil" },
		{ "Forceadd","forceadd", "!= nil" },
	*/

	compatList, ok := HeaderValidationMap[sType]
	if !ok {
		return errors.Errorf("Unknown set type %s", sType)
	}

	if header.Family != "" {
		if !strings.Contains(compatList, "-family-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "family")
		}

	}

	if header.Range != "" {
		if !strings.Contains(compatList, "-range-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "range")
		}

	}

	if header.Hashsize != 0 {
		if !strings.Contains(compatList, "-hashsize-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "hashsize")
		}

	}

	if header.Maxelem != 0 {
		if !strings.Contains(compatList, "-maxelem-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "maxelem")
		}

	}

	if header.References != 0 {
		if !strings.Contains(compatList, "-references-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "references")
		}

	}

	if header.Timeout != 0 {
		if !strings.Contains(compatList, "-timeout-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "timeout")
		}

	}

	if header.Netmask != 0 {
		if !strings.Contains(compatList, "-netmask-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "netmask")
		}

	}

	if header.Size != 0 {
		if !strings.Contains(compatList, "-size-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "size")
		}

	}

	if header.Counters != nil {
		if !strings.Contains(compatList, "-counters-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "counters")
		}

	}

	if header.Comment != nil {
		if !strings.Contains(compatList, "-comment-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "comment")
		}

	}

	if header.SKBInfo != nil {
		if !strings.Contains(compatList, "-skbinfo-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "skbinfo")
		}

	}

	if header.Forceadd != nil {
		if !strings.Contains(compatList, "-forceadd-") {
			return errors.Errorf("Set of Type %s incompatible with header %s", sType, "forceadd")
		}

	}

	return nil
}

var (
	HeaderValidationMap = map[SetType]string{
		SetBitmapIp:       "-range-netmask-timeout-counters-comment-skbinfo-",
		SetBitmapIpMac:    "-range-timeout-counters-comment-skbinfo-",
		SetBitmapPort:     "-range-timeout-counters-comment-skbinfo-",
		SetHashIp:         "-family-hashsize-maxelem-netmask-timeout-counters-comment-skbinfo-forceadd-",
		SetHashMac:        "-hashsize-maxelem-timeout-counters-comment-skbinfo-forceadd-",
		SetHashNet:        "-family-hashsize-maxelem-timeout-counters-comment-skbinfo-forceadd-",
		SetHashNetNet:     "-family-hashsize-maxelem-timeout-counters-comment-skbinfo-forceadd-",
		SetHashIpPort:     "-family-hashsize-maxelem-timeout-counters-comment-skbinfo-forceadd-",
		SetHashNetPort:    "-family-hashsize-maxelem-timeout-counters-comment-skbinfo-forceadd-",
		SetHashIpPortIp:   "-family-hashsize-maxelem-timeout-counters-comment-skbinfo-forceadd-",
		SetHashIpPortNet:  "-family-hashsize-maxelem-timeout-counters-comment-skbinfo-forceadd-",
		SetHashIpMark:     "-family-markmask-hashsize-maxelem-timeout-counters-comment-skbinfo-forceadd-",
		SetHashNetPortNet: "-family-hashsize-maxelem-timeout-counters-comment-skbinfo-forceadd-",
		SetHashNetIface:   "-family-hashsize-maxelem-timeout-counters-comment-skbinfo-forceadd-",
		SetListSet:        "-size-timeout-counters-comment-skbinfo-",
	}
)

var (
	// NoVal used in Header and Member structs as a value for fields
	// which don't have value. Like Header.Comment or Member.NoMatch.
	// This is the artifact of xml parsing.
	NoVal = new(string)
)
