package ipset

import "fmt"

type Set struct {
	Name     string   `xml:" name,attr"  json:",omitempty"`
	Header   *Header  `xml:" header,omitempty" json:"header,omitempty"`
	Members  []Member `xml:" members>member,omitempty" json:"members,omitempty"`
	Revision int      `xml:" revision,omitempty" json:"revision,omitempty"`
	Type     string   `xml:" type,omitempty" json:"type,omitempty"`
}

func (s *Set) Render(rType RenderType) string {
	var result string

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
	// only flush set
	case RenderFlush:
		result += fmt.Sprintf("flush %s\n", s.Name)
	// only destroy set
	case RenderDestroy:
		result += fmt.Sprintf("destroy %s\n", s.Name)
	}

	return result
}

type SetType string

const (
	SetBitmapIp       = "bitmap:ip"
	SetBitmapIpMac    = "bitmap:ip,mac"
	SetBitmapIpPort   = "bitmap:port"
	SetHashIp         = "hash:ip"
	SetHashMac        = "hash:mac"
	SetHashNet        = "hash:net"
	SetHashNetNet     = "hash:net,net"
	SetHadhIpPort     = "hash:ip,port"
	SetHashNetPort    = "hash:net,port"
	SetHashIpPortIp   = "hash:ip,port,ip"
	SetHashIpPortNet  = "hash:ip,port,net"
	SetHashIpMark     = "hash:ip,mark"
	SetHashNetPortNet = "hash:net,port,net"
	SetHashNetIface   = "hash:net,iface"
	SetListSet        = "list:set"
)
