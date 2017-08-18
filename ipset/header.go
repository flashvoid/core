package ipset

import "fmt"

type Header struct {
	Family     string  `xml:" family,omitempty" json:"family,omitempty"`
	Range      string  `xml:" range,omitempty" json:"range,omitempty"`
	Hashsize   int     `xml:" hashsize,omitempty" json:"hashsize,omitempty"`
	Maxelem    int     `xml:" maxelem,omitempty" json:"maxelem,omitempty"`
	Memsize    int     `xml:" memsize,omitempty" json:"memsize,omitempty"`
	References int     `xml:" references,omitempty" json:"references,omitempty"`
	Timeout    int     `xml:" timeout,omitempty" json:"timeout,omitempty"`
	Netmask    int     `xml:" netmask,omitempty" json:"netmask,omitempty"`
	Counters   *string `xml:" counters,omitempty" json:"counters,omitempty"`
	Comment    *string `xml:" comment,omitempty" json:"comment,omitempty"`
	SKBInfo    *string `xml:" skbinfo,omitempty" json:"skbinfo,omitempty"`
	Forceadd   *string `xml:" forceadd,omitempty" json:"forceadd,omitempty"`
}

func (h *Header) render() string {
	var result string
	if h == nil {
		return result
	}

	if h.Family != "" {
		result += fmt.Sprintf(" family %s", h.Family)
	}

	if h.Range != "" {
		result += fmt.Sprintf(" range %s", h.Range)
	}

	if h.Hashsize != 0 {
		result += fmt.Sprintf(" hashsize %d", h.Hashsize)
	}

	if h.Maxelem != 0 {
		result += fmt.Sprintf(" maxelem %d", h.Maxelem)
	}

	if h.Memsize != 0 {
		result += fmt.Sprintf(" memsize %d", h.Memsize)
	}

	if h.References != 0 {
		result += fmt.Sprintf(" references %d", h.References)
	}

	if h.Timeout != 0 {
		result += fmt.Sprintf(" timeout %d", h.Timeout)
	}

	if h.Netmask != 0 {
		result += fmt.Sprintf(" netmask %d", h.Netmask)
	}

	if h.Counters != nil {
		result += fmt.Sprintf(" counters")
	}

	if h.Comment != nil {
		result += fmt.Sprintf(" comment")
	}

	if h.SKBInfo != nil {
		result += fmt.Sprintf(" skbinfo")
	}

	if h.Forceadd != nil {
		result += fmt.Sprintf(" forceadd")
	}

	return result

}
