package dns

import (
	"net"

	miekg "github.com/miekg/dns"
)

// typeA writer returns a static A record resolving names to localhost
func ARecordResponseLoopback(w miekg.ResponseWriter, req *miekg.Msg) {
	m := new(miekg.Msg)
	m.SetReply(req)

	m.Answer = []miekg.RR{
		&miekg.A{
			Hdr: miekg.RR_Header{
				Name:   m.Question[0].Name,
				Rrtype: miekg.TypeA,
				Class:  miekg.ClassINET,
				Ttl:    0,
			},
			A: net.IP{127, 0, 0, 1},
		},
	}
	_ = w.WriteMsg(m)
}

// typeA writer returns a static A record resolving names to localhost
func AAAARecordResponseLoopback(w miekg.ResponseWriter, req *miekg.Msg) {
	m := new(miekg.Msg)
	m.SetReply(req)

	m.Answer = []miekg.RR{
		&miekg.AAAA{
			Hdr: miekg.RR_Header{
				Name:   m.Question[0].Name,
				Rrtype: miekg.TypeAAAA,
				Class:  miekg.ClassINET,
				Ttl:    0,
			},
			AAAA: net.IPv6loopback,
		},
	}
	_ = w.WriteMsg(m)
}

// typeA writer returns a static A record resolving names to localhost
func CNAMERecordResponse(w miekg.ResponseWriter, req *miekg.Msg) {
	m := new(miekg.Msg)
	m.SetReply(req)

	m.Ns = []miekg.RR{
		&miekg.NS{
			Hdr: miekg.RR_Header{
				Name:   m.Question[0].Name,
				Rrtype: miekg.TypeCNAME,
				Class:  miekg.ClassINET,
				Ttl:    300,
			},
			Ns: "infrabin.com.",
		},
	}
	_ = w.WriteMsg(m)
}
