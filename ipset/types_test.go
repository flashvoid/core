package ipset

import (
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
