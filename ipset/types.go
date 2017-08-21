package ipset

import "fmt"

type Ipset struct {
	Sets []*Set `xml:" ipset,omitempty" json:"ipset,omitempty"`
}

func (s *Ipset) SetByName(name string) *Set {
	for i, set := range s.Sets {
		if set.Name == name {
			return s.Sets[i]
		}
	}
	return nil
}

func (s *Ipset) Render(rType RenderType) string {
	var result string

	switch rType {
	case RenderFlush:
		if len(s.Sets) == 0 {
			return "flush\n"
		}
	case RenderDestroy:
		if len(s.Sets) == 0 {
			return "destroy\n"
		}

	// RenderSwap will fail when Sets < 2,
	// it's a duty of a caller to ensure
	// correctness.
	case RenderSwap:
		return fmt.Sprintf("swap %s %s", s.Sets[0].Name, s.Sets[1].Name)

	// RenderRename will fail when Sets < 2,
	// it's a duty of a caller to ensure
	// correctness.
	case RenderRename:
		return fmt.Sprintf("rename %s %s", s.Sets[0].Name, s.Sets[1].Name)
	}

	for _, set := range s.Sets {
		result += set.Render(rType)
	}

	return result
}

type RenderType int

const (
	// Render all sets and members for creation/addition.
	// Same as regular save.
	RenderSave RenderType = iota

	// Render all sets for creation.
	RenderCreate

	// Render all members for addition.
	RenderAdd

	// Render all members for deletion.
	RenderDelete

	// Render all sets for deletion.
	RenderFlush

	// Render all sets for destroy.
	RenderDestroy

	// Render set for swap
	RenderSwap

	// Render set for test
	RenderTest

	// Render set for rename
	RenderRename
)
