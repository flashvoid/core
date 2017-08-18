package ipset

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	cases := []struct {
		name   string
		sets   Ipset
		rType  RenderType
		expect string
	}{
		{
			name: "Render set and members",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: "hash:net", Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar"}}},
			}},
			rType:  RenderSave,
			expect: "create super hash:net \nadd super foo\nadd super bar\n",
		},
		{
			name: "Render set with header",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: "hash:net", Header: &Header{Family: "inet", Hashsize: 4}},
			}},
			rType:  RenderSave,
			expect: "create super hash:net  family inet hashsize 4\n",
		},
		{
			name: "Render set members for creation",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: "hash:net", Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar"}}},
			}},
			rType:  RenderAdd,
			expect: "add super foo\nadd super bar\n",
		},
		{
			name: "Render set members for deleteion",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: "hash:net", Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar"}}},
			}},
			rType:  RenderDelete,
			expect: "del super foo\ndel super bar\n",
		},
		{
			name: "Render sets for flush",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: "hash:net", Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar"}}},
			}},
			rType:  RenderFlush,
			expect: "flush super\n",
		},
		{
			name: "Render sets for flush",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: "hash:net", Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar"}}},
			}},
			rType:  RenderDestroy,
			expect: "destroy super\n",
		},
		{
			name:   "Render flush all",
			sets:   Ipset{},
			rType:  RenderDestroy,
			expect: "destroy\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := tc.sets.Render(tc.rType)
			if res != tc.expect {
				t.Fatalf("Expected:\n%s\ngot:\n%s\n", []byte(tc.expect), []byte(res))
			}
		})
	}
}

func TestNewMember(t *testing.T) {
	cases := []struct {
		name   string
		set    Set
		elem   string
		args   []MemberOpt
		expect func(*Member, error) bool
	}{
		{
			name:   "make simple member",
			set:    Set{Name: "super", Type: SetHashNet},
			elem:   "foo",
			args:   []MemberOpt{OptMemComment("test")},
			expect: func(m *Member, e error) bool { return m.Comment == "test" && e == nil },
		},
		{
			name:   "error when set doesn't allow comments",
			set:    Set{Name: "super", Type: SetHashNet, Header: &Header{}},
			elem:   "foo",
			args:   []MemberOpt{OptMemComment("test")},
			expect: func(m *Member, e error) bool { return e.Error() == "comment options used with incompatible set" },
		},
		{
			name:   "set allows comments",
			set:    Set{Name: "super", Type: SetHashNet, Header: &Header{Comment: new(string)}},
			elem:   "foo",
			args:   []MemberOpt{OptMemComment("test")},
			expect: func(m *Member, e error) bool { return m.Comment == "test" && e == nil },
		},
		{
			name:   "error when set doesn't allow timeouts",
			set:    Set{Name: "super", Type: SetHashNet, Header: &Header{}},
			elem:   "foo",
			args:   []MemberOpt{OptMemTimeout(10)},
			expect: func(m *Member, e error) bool { return e.Error() == "timeout options used with incompatible set" },
		},
		{
			name:   "set allows timeouts",
			set:    Set{Name: "super", Type: SetHashNet, Header: &Header{Timeout: 1}},
			elem:   "foo",
			args:   []MemberOpt{OptMemTimeout(10)},
			expect: func(m *Member, e error) bool { return m.Timeout == 10 && e == nil },
		},
		{
			name:   "error when set doesn't allow counters",
			set:    Set{Name: "super", Type: SetHashNet, Header: &Header{}},
			elem:   "foo",
			args:   []MemberOpt{OptMemBytes(10), OptMemPackets(2)},
			expect: func(m *Member, e error) bool { return e.Error() == "bytes options used with incompatible set" },
		},
		{
			name:   "set allows counters",
			set:    Set{Name: "super", Type: SetHashNet, Header: &Header{Counters: new(string)}},
			elem:   "foo",
			args:   []MemberOpt{OptMemBytes(10), OptMemPackets(2)},
			expect: func(m *Member, e error) bool { return m.Bytes == 10 && e == nil },
		},
		{
			name:   "error when set doesn't allow skbinfo",
			set:    Set{Name: "super", Type: SetHashNet, Header: &Header{}},
			elem:   "foo",
			args:   []MemberOpt{OptMemSKBPrio("10")},
			expect: func(m *Member, e error) bool { return e.Error() == "skbprio options used with incompatible set" },
		},
		{
			name:   "set allows skbinfo",
			set:    Set{Name: "super", Type: SetHashNet, Header: &Header{SKBInfo: new(string)}},
			elem:   "foo",
			args:   []MemberOpt{OptMemSKBPrio("10")},
			expect: func(m *Member, e error) bool { return m.SKBPrio == "10" && e == nil },
		},
		{
			name:   "test nomatch",
			set:    Set{Name: "super", Type: SetHashNet},
			elem:   "foo",
			args:   []MemberOpt{OptMemNomatch},
			expect: func(m *Member, e error) bool { return *m.NoMatch == "" && e == nil },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewMember(tc.elem, &tc.set, tc.args...)
			if !tc.expect(m, err) {
				t.Fatalf("Unexpected NewMember() %+v, %s", m, err)
			}
		})
	}
}

func TestNewSet(t *testing.T) {
	cases := []struct {
		name    string
		setName string
		setType SetType
		args    []SetOpt
		expect  func(*Set, error) bool
	}{
		{
			name:    "make simple set",
			setName: "super",
			setType: SetHashNet,
			args:    []SetOpt{OptSetFamily("inet"), OptSetRevision(1)},
			expect:  func(s *Set, e error) bool { return s.Header.Family == "inet" && e == nil },
		},
		{
			name:    "test set size",
			setName: "super",
			setType: SetListSet,
			args:    []SetOpt{OptSetSize(4)},
			expect:  func(s *Set, e error) bool { return s.Header.Size == 4 && e == nil },
		},
		{
			name:    "test set size with error",
			setName: "super",
			setType: SetHashNet,
			args:    []SetOpt{OptSetSize(4)},
			expect:  func(s *Set, e error) bool { return strings.Contains(e.Error(), "incompatible") },
		},
		{
			name:    "test set range",
			setName: "super",
			setType: SetBitmapIp,
			args:    []SetOpt{OptSetRange("192.168.0.0/16")},
			expect:  func(s *Set, e error) bool { return s.Header.Range == "192.168.0.0/16" && e == nil },
		},
		{
			name:    "test set range with error",
			setName: "super",
			setType: SetHashNet,
			args:    []SetOpt{OptSetRange("192.168.0.0/16")},
			expect:  func(s *Set, e error) bool { return strings.Contains(e.Error(), "incompatible") },
		},
		{
			name:    "test hashsize",
			setName: "super",
			setType: SetHashNet,
			args:    []SetOpt{OptSetHashsize(4)},
			expect:  func(s *Set, e error) bool { return s.Header.Hashsize == 4 && e == nil },
		},
		{
			name:    "test hashsize with error",
			setName: "super",
			setType: SetBitmapIp,
			args:    []SetOpt{OptSetHashsize(4)},
			expect:  func(s *Set, e error) bool { return strings.Contains(e.Error(), "incompatible") },
		},
		{
			name:    "test maxelem",
			setName: "super",
			setType: SetHashNet,
			args:    []SetOpt{OptSetMaxelem(4)},
			expect:  func(s *Set, e error) bool { return s.Header.Maxelem == 4 && e == nil },
		},
		{
			name:    "test maxelem with error",
			setName: "super",
			setType: SetBitmapIp,
			args:    []SetOpt{OptSetMaxelem(4)},
			expect:  func(s *Set, e error) bool { return strings.Contains(e.Error(), "incompatible") },
		},
		{
			name:    "test maxelem",
			setName: "super",
			setType: SetHashNet,
			args:    []SetOpt{OptSetMaxelem(4)},
			expect:  func(s *Set, e error) bool { return s.Header.Maxelem == 4 && e == nil },
		},
		{
			name:    "test maxelem with error",
			setName: "super",
			setType: SetBitmapIp,
			args:    []SetOpt{OptSetMaxelem(4)},
			expect:  func(s *Set, e error) bool { return strings.Contains(e.Error(), "incompatible") },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewSet(tc.setName, tc.setType, tc.args...)
			if !tc.expect(s, err) {
				ensureHeader(s)
				t.Fatalf("Unexpected NewSet() %+v, %+v, %s", s, s.Header, err)
			}
		})
	}
}
