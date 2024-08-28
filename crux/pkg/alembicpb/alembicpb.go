package alembicpb

import (
	"fmt"
)

var lval struct {
	str string
	num int
}

// ProtoPkg is what we extract from a proto file
type ProtoPkg struct {
	Pre      string
	Package  string
	Services []*ProtoService
}

// ProtoService is a protobuf service
type ProtoService struct {
	Name string
	RPC  []ProtoRPC
}

// ProtoRPC is a single rpc
type ProtoRPC struct {
	Name       string
	Req, Reply ProtoTraffic
}

// ProtoTraffic is traffic
type ProtoTraffic struct {
	Name      string
	Streaming bool
}

// FindRPC err, finds a given rpc in a given service
func (p *ProtoPkg) FindRPC(svc, rpc string) *ProtoRPC {
	for _, s := range p.Services {
		if s.Name == svc {
			for _, r := range s.RPC {
				if r.Name == rpc {
					return &r
				}
			}
		}
	}
	return nil
}

func (p *ProtoPkg) String() string {
	s := "{\n"
	s += fmt.Sprintf("\tPre = '%s'\n", p.Pre)
	for _, sv := range p.Services {
		s += sv.String("\t")
	}
	s += "}\n"
	return s
}

func (s *ProtoService) String(pre string) string {
	str := fmt.Sprintf("%sservice %s {\n", pre, s.Name)
	for _, r := range s.RPC {
		str += fmt.Sprintf("%s\t%s\n", pre, r.String())
	}
	str += fmt.Sprintf("%s}\n", pre)
	return str
}

func (r *ProtoRPC) String() string {
	return fmt.Sprintf("rpc %s(%s) returns (%s) {}", r.Name, r.Req.String(), r.Reply.String())
}

func (t *ProtoTraffic) String() string {
	var s string
	if t.Streaming {
		s += "stream "
	}
	return s + t.Name
}
