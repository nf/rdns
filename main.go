// rdns is a DNS server that serves PTR and AAAA records for an IPv6 subnet, as
// well as the associated authoritative NS records.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"

	"github.com/miekg/dns"
)

var (
	networkFlag    = flag.String("network", "fe80::/64", "`subnet` for which to serve ip6.arpa records")
	hostPrefixFlag = flag.String("host_prefix", "ip-", "prefix for generated host name")
	domainFlag     = flag.String("domain", ".v6.example.com.", "domain suffix for generated domain name")
	nsFlag         = flag.String("ns", "ns.example.com.", "name `server` for NS responses")
	ttlFlag        = flag.Int("ttl", 3600, "answer TTL in `seconds`")
	addrFlag       = flag.String("listen", ":dns", "DNS server listen `address`")
)

func main() {
	flag.Parse()

	_, network, err := net.ParseCIDR(*networkFlag)
	if err != nil {
		log.Fatalf("bad -network flag: %v", err)
	}
	ones, _ := network.Mask.Size()
	if ones%8 != 0 {
		log.Fatalf("bad -network flag: mask must be a multiple of 8")
	}
	prefix := fmt.Sprintf("%x", []byte(network.IP[:ones/8]))

	s := &server{
		prefix: prefix,
		ns:     regexp.MustCompile(`^((?:[[:xdigit:]]\.){` + fmt.Sprint(len(prefix)) + `,32})ip6\.arpa\.$`),
		ptr:    regexp.MustCompile(`^((?:[[:xdigit:]]\.){32})ip6\.arpa\.$`),
		aaaa: regexp.MustCompile(`^` +
			regexp.QuoteMeta(*hostPrefixFlag) +
			`([[:xdigit:]]{1,` + fmt.Sprint(32-len(prefix)) + `})` +
			regexp.QuoteMeta(*domainFlag) +
			`$`),
	}
	log.Fatal((&dns.Server{
		Addr:    *addrFlag,
		Net:     "udp",
		Handler: s,
	}).ListenAndServe())
}

type server struct {
	prefix        string
	ns, ptr, aaaa *regexp.Regexp
}

func (s *server) ServeDNS(w dns.ResponseWriter, m *dns.Msg) {
	if len(m.Question) != 1 {
		log.Printf("len(question) = %d", len(m.Question))
		return
	}
	q := m.Question[0]

	reply := new(dns.Msg)
	reply.SetReply(m)

	switch q.Qtype {
	case dns.TypeNS:
		s.toNS(reply, q.Name)
	case dns.TypePTR:
		s.toPTR(reply, q.Name)
	case dns.TypeAAAA:
		s.toAAAA(reply, q.Name)
	}

	var h *dns.RR_Header
	if len(reply.Answer) == 1 {
		h = reply.Answer[0].Header()
	} else if len(reply.Ns) == 1 {
		h = reply.Ns[0].Header()
	}
	if h != nil {
		reply.Authoritative = true
		h.Class = dns.ClassINET
		h.Name = q.Name
		h.Ttl = uint32(*ttlFlag)
	} else {
		reply.Rcode = dns.RcodeNameError
	}

	log.Printf("reply to %v:\n%v", w.RemoteAddr(), reply)
	if err := w.WriteMsg(reply); err != nil {
		log.Printf("WriteMsg error: %v", err)
	}
}

func (s *server) toNS(reply *dns.Msg, name string) {
	if _, ok := s.hexMatch(s.ns, name); !ok {
		return
	}
	reply.Ns = []dns.RR{&dns.NS{
		Hdr: dns.RR_Header{
			Rrtype:   dns.TypeNS,
			Rdlength: uint16(len(*nsFlag)),
		},
		Ns: *nsFlag,
	}}
}

func (s *server) toPTR(reply *dns.Msg, name string) {
	x, ok := s.hexMatch(s.ptr, name)
	if !ok {
		return
	}
	x = strings.TrimLeft(x, "0")
	ptr := *hostPrefixFlag + x + *domainFlag
	reply.Answer = []dns.RR{&dns.PTR{
		Hdr: dns.RR_Header{
			Rrtype:   dns.TypePTR,
			Rdlength: uint16(len(ptr)),
		},
		Ptr: ptr,
	}}
}

func (s *server) hexMatch(re *regexp.Regexp, name string) (suffix string, _ bool) {
	m := re.FindStringSubmatch(name)
	if m == nil {
		return "", false
	}
	d := strings.Split(m[1][:len(m[1])-1], ".")
	n := len(d)
	for i := 0; i < n/2; i++ {
		d[i], d[n-1-i] = d[n-1-i], d[i]
	}
	x := strings.Join(d, "")
	if !strings.HasPrefix(x, s.prefix) {
		return "", false
	}
	return x[len(s.prefix):], true
}

func (s *server) toAAAA(reply *dns.Msg, name string) {
	m := s.aaaa.FindStringSubmatch(name)
	if m == nil {
		return
	}
	x := m[1]
	if pad := 32 - len(x) - len(s.prefix); pad > 0 {
		x = strings.Repeat("0", pad) + x
	}
	ip, err := hex.DecodeString(s.prefix + x)
	if err != nil {
		return
	}
	aaaa := net.IP(ip)
	reply.Answer = []dns.RR{&dns.AAAA{
		Hdr: dns.RR_Header{
			Rrtype:   dns.TypeAAAA,
			Rdlength: uint16(len(aaaa)),
		},
		AAAA: aaaa,
	}}
}
