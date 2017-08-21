package ipset

import (
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestRender(t *testing.T) {
	cases := []struct {
		name   string
		sets   Ipset
		rType  RenderType
		expect string
	}{
		{
			name: "Render set and member for test",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashNet, Members: []Member{Member{Elem: "foo"}}},
			}},
			rType:  RenderTest,
			expect: "test super foo",
		},
		{
			name: "Render set  for test",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashNet},
			}},
			rType:  RenderTest,
			expect: "test super",
		},
		{
			name: "Render set and members",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashNet, Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar"}}},
			}},
			rType:  RenderSave,
			expect: "create super hash:net \nadd super foo\nadd super bar\n",
		},
		{
			name: "Render set and members with counters",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashIp, Header: Header{Counters: new(string)},
					Members: []Member{Member{Elem: "foo", Packets: 42, Bytes: 1024}}},
			}},
			rType:  RenderSave,
			expect: "create super hash:ip  counters\nadd super foo packets 42 bytes 1024\n",
		},
		{
			name: "Render set and members with timeout",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashIp, Header: Header{Timeout: 600},
					Members: []Member{Member{Elem: "foo", Timeout: 300}, Member{Elem: "bar"}}},
			}},
			rType:  RenderSave,
			expect: "create super hash:ip  timeout 600\nadd super foo timeout 300\nadd super bar\n",
		},
		{
			name: "Render set and members with nomatch",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashNet,
					Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar", NoMatch: new(string)}}},
			}},
			rType:  RenderSave,
			expect: "create super hash:net \nadd super foo\nadd super bar nomatch\n",
		},
		{
			name: "Render set and members with comment",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashIp, Header: Header{Comment: new(string)},
					Members: []Member{Member{Elem: "foo", Comment: "allow access to SMB share on fileserv"}}},
			}},
			rType:  RenderSave,
			expect: "create super hash:ip  comment\nadd super foo comment allow access to SMB share on fileserv\n",
		},
		{
			name: "Render set and members with skb",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashIp, Header: Header{SKBInfo: new(string)},
					Members: []Member{Member{Elem: "foo", SKBMark: "0x1111/0xff00ffff", SKBPrio: "1:10", SKBQueue: "10"}}},
			}},
			rType:  RenderSave,
			expect: "create super hash:ip  skbinfo\nadd super foo skbmark 0x1111/0xff00ffff skbprio 1:10 skbqueue 10\n",
		},
		{
			name: "Render set and members with hashsize",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashIp, Header: Header{Hashsize: 1536}}}},
			rType:  RenderSave,
			expect: "create super hash:ip  hashsize 1536\n",
		},
		{
			name: "Render set and members with maxelem",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashIp, Header: Header{Maxelem: 2048}}}},
			rType:  RenderSave,
			expect: "create super hash:ip  maxelem 2048\n",
		},
		{
			name: "Render set and members with family",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashIp, Header: Header{Family: MemberFamilyInet6}}}},
			rType:  RenderSave,
			expect: "create super hash:ip  family inet6\n",
		},
		{
			name: "Render set and members with forceadd",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashIp, Header: Header{Forceadd: NoVal}}}},
			rType:  RenderSave,
			expect: "create super hash:ip  forceadd\n",
		},
		{
			name: "Render set with size",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetListSet, Header: Header{Size: 8}}}},
			rType:  RenderSave,
			expect: "create super list:set  size 8\n",
		},
		{
			name: "Render set and members with range",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashIp, Header: Header{Range: "192.168.0.0/16"}}}},
			rType:  RenderSave,
			expect: "create super hash:ip  range 192.168.0.0/16\n",
		},
		{
			name: "Render set and members with range and mac",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetBitmapIpMac, Header: Header{Range: "192.168.0.0/16"},
					Members: []Member{Member{Elem: "192.168.1/24"}}},
			}},
			rType:  RenderSave,
			expect: "create super bitmap:ip,mac  range 192.168.0.0/16\nadd super 192.168.1/24\n",
		},
		{
			name: "Render set and members with range and port",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetBitmapPort, Header: Header{Range: "0-1024"},
					Members: []Member{Member{Elem: "80"}}},
			}},
			rType:  RenderSave,
			expect: "create super bitmap:port  range 0-1024\nadd super 80\n",
		},
		{
			name: "Render set and members with netmask",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashIp, Header: Header{Netmask: 30},
					Members: []Member{Member{Elem: "192.168.1.0/24"}}},
			}},
			rType:  RenderSave,
			expect: "create super hash:ip  netmask 30\nadd super 192.168.1.0/24\n",
		},
		{
			name: "Render set with header",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashNet, Header: Header{Family: "inet", Hashsize: 4}},
			}},
			rType:  RenderSave,
			expect: "create super hash:net  family inet hashsize 4\n",
		},
		{
			name: "Render set members for creation",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashNet, Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar"}}},
			}},
			rType:  RenderAdd,
			expect: "add super foo\nadd super bar\n",
		},
		{
			name: "Render set members for deleteion",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashNet, Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar"}}},
			}},
			rType:  RenderDelete,
			expect: "del super foo\ndel super bar\n",
		},
		{
			name: "Render sets for flush",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashNet, Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar"}}},
			}},
			rType:  RenderFlush,
			expect: "flush super\n",
		},
		{
			name: "Render sets for destroy",
			sets: Ipset{Sets: []*Set{
				&Set{Name: "super", Type: SetHashNet, Members: []Member{Member{Elem: "foo"}, Member{Elem: "bar"}}},
			}},
			rType:  RenderDestroy,
			expect: "destroy super\n",
		},
		{
			name:   "Render flush all",
			sets:   Ipset{},
			rType:  RenderFlush,
			expect: "flush\n",
		},
		{
			name:   "Render destroy all",
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
			name:   "error when set doesn't allow comments",
			set:    Set{Name: "super", Type: SetHashNet, Header: Header{}},
			elem:   "foo",
			args:   []MemberOpt{OptMemComment("test")},
			expect: func(m *Member, e error) bool { return e.Error() == "comment options used with incompatible set" },
		},
		{
			name:   "set allows comments",
			set:    Set{Name: "super", Type: SetHashNet, Header: Header{Comment: new(string)}},
			elem:   "foo",
			args:   []MemberOpt{OptMemComment("test")},
			expect: func(m *Member, e error) bool { return m.Comment == "test" && e == nil },
		},
		{
			name:   "error when set doesn't allow timeouts",
			set:    Set{Name: "super", Type: SetHashNet, Header: Header{}},
			elem:   "foo",
			args:   []MemberOpt{OptMemTimeout(10)},
			expect: func(m *Member, e error) bool { return e.Error() == "timeout options used with incompatible set" },
		},
		{
			name:   "set allows timeouts",
			set:    Set{Name: "super", Type: SetHashNet, Header: Header{Timeout: 1}},
			elem:   "foo",
			args:   []MemberOpt{OptMemTimeout(10)},
			expect: func(m *Member, e error) bool { return m.Timeout == 10 && e == nil },
		},
		{
			name:   "error when set doesn't allow counters",
			set:    Set{Name: "super", Type: SetHashNet, Header: Header{}},
			elem:   "foo",
			args:   []MemberOpt{OptMemBytes(10), OptMemPackets(2)},
			expect: func(m *Member, e error) bool { return e.Error() == "bytes options used with incompatible set" },
		},
		{
			name:   "set allows counters",
			set:    Set{Name: "super", Type: SetHashNet, Header: Header{Counters: new(string)}},
			elem:   "foo",
			args:   []MemberOpt{OptMemBytes(10), OptMemPackets(2)},
			expect: func(m *Member, e error) bool { return m.Bytes == 10 && e == nil },
		},
		{
			name:   "error when set doesn't allow skbinfo",
			set:    Set{Name: "super", Type: SetHashNet, Header: Header{}},
			elem:   "foo",
			args:   []MemberOpt{OptMemSKBPrio("10")},
			expect: func(m *Member, e error) bool { return e.Error() == "skbprio options used with incompatible set" },
		},
		{
			name:   "set allows skbinfo",
			set:    Set{Name: "super", Type: SetHashNet, Header: Header{SKBInfo: new(string)}},
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
			spew.Dump(m)
			t.Log(err)
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
			name:    "test netmask with error",
			setName: "super",
			setType: SetHashNet,
			args:    []SetOpt{OptSetNetmask(30)},
			expect:  func(s *Set, e error) bool { return strings.Contains(e.Error(), "incompatible") },
		},
		{
			name:    "test netmask",
			setName: "super",
			setType: SetBitmapIp,
			args:    []SetOpt{OptSetNetmask(30)},
			expect:  func(s *Set, e error) bool { return s.Header.Netmask == 30 && e == nil },
		},
		{
			name:    "test forceadd with error",
			setName: "super",
			setType: SetBitmapIp,
			args:    []SetOpt{OptSetForceadd()},
			expect:  func(s *Set, e error) bool { return strings.Contains(e.Error(), "incompatible") },
		},
		{
			name:    "test forceadd",
			setName: "super",
			setType: SetHashNet,
			args:    []SetOpt{OptSetForceadd()},
			expect:  func(s *Set, e error) bool { return s.Header.Forceadd == NoVal && e == nil },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewSet(tc.setName, tc.setType, tc.args...)
			if !tc.expect(s, err) {
				if s != nil {
					t.Logf("Header: %+v", s.Header)
				}
				t.Fatalf("Unexpected NewSet() %+v, %s", s, err)
			}
		})
	}
}
