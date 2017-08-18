package ipset

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

	if len(s.Sets) == 0 {
		switch rType {
		case RenderFlush:
			return "flush\n"
		case RenderDestroy:
			return "destroy\n"
		}
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
)
