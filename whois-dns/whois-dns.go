package main

import (
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

var (
	httpPort      string
	dnsPort       string
	dnsTCP        bool
	dnsA          string
	dnsARecord    net.IP
	dnsHostLimit  int64
	dnsHostExpire int64
	verbose       bool
)

func Verbose(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}

func init() {
	flag.StringVar(&httpPort, "http-port", ":80", "Specify the HTTP server port")
	flag.StringVar(&dnsPort, "dns-port", ":53", "Specify the DNS server port")
	flag.StringVar(&dnsA, "dns-A", "", "The A record returned by the DNS server")
	flag.Int64Var(&dnsHostLimit, "dns-host-limit", 5*100000, "Maximum DNS record number")
	flag.Int64Var(&dnsHostExpire, "dns-host-expire", 5, "Generated record expiration time (second)")
	flag.BoolVar(&dnsTCP, "dns-tcp", false, "Support TCP for DNS server")
	flag.BoolVar(&verbose, "verbose", false, "")
}

var dnsHostCount int64

func HostCounterIncrease() bool {
	if dnsHostCount+1 > dnsHostLimit {
		return false
	}
	dnsHostCount = atomic.AddInt64(&dnsHostCount, 1)
	return true
}

func HostCounterReduce() bool {
	if dnsHostCount == 0 {
		return false
	}
	dnsHostCount = atomic.AddInt64(&dnsHostCount, -1)
	return true
}

func main() {
	flag.Parse()

	Verbose("flag parse done")

	if dnsA != "" {
		dnsARecord = net.ParseIP(dnsA)
	}

	var err error
	if dnsARecord == nil {
		dnsARecord, err = getInterfaceAddr()
		if err != nil {
			log.Fatal(err)
		}
	}

	if dnsARecord == nil {
		log.Fatal("no available address for an A record")
	}

	Verbose("dns A Record:", dnsARecord)

	SetHandles()
	go func() {
		if err := dns.ListenAndServe(dnsPort, "udp", nil); err != nil {
			log.Fatalln(err)
		}
	}()

	if dnsTCP {
		go func() {
			if err := dns.ListenAndServe(dnsPort, "tcp", nil); err != nil {
				log.Fatalln(err)
			}
		}()
	}

	go func() {
		for {
			t := time.Now().Unix()
			validHost.Range(func(key, value interface{}) bool {
				rcd := value.(Record)
				if t-rcd.ttl > dnsHostExpire {
					Verbose("dns zone delete: ", key, value)
					validHost.Delete(key)
					HostCounterReduce()
				}
				return true
			})

			time.Sleep(5 * time.Second)
		}
	}()

	log.Fatalln(http.ListenAndServe(httpPort, nil))
}

func getInterfaceAddr() (net.IP, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, v := range ifs {
		addrs, err := v.Addrs()
		if err != nil {
			log.Fatal(err)
		}

		if v.Name == "lo" ||
			len(addrs) == 0 {
			continue
		}

		addr, _, err := net.ParseCIDR(addrs[0].String())
		return addr, err
	}

	return nil, nil
}

type Record struct {
	HTTPRemoteAddr   string `json:"http_remote_addr,omitempty"`
	DNSRemoteAddr    string `json:"dns_remote_addr,omitempty"`
	EDNSClientSubnet string `json:"edns-client-subnet,omitempty"`
	ttl              int64  `json:"ttl,omitempty"`
}

var validHost sync.Map

func generateHost(r *http.Request) string {
	host := r.Host
	addr := r.RemoteAddr
	id := rand.NewSource(time.Now().UnixNano()).Int63()
	sin := md5.Sum([]byte(strconv.Itoa(int(id)) + addr))
	return fmt.Sprintf("%x.%s", sin, host)
}

func parseAddr(addr string) net.IP {
	i := strings.LastIndex(addr, ":")
	ip := addr[0:i]
	return net.ParseIP(ip)
}

func SetHandles() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		Verbose("http req:", r.Method, r.RequestURI)

		host := generateHost(r)
		scheme := r.URL.Scheme
		if scheme == "" {
			scheme = "http"
		}

		if !HostCounterIncrease() {
			Verbose("dns host overflow", dnsHostLimit)
			http.NotFound(w, r)
			return
		}

		redirectURL := fmt.Sprintf("%s://%s/feedback", scheme, host)

		rcd := Record{
			ttl: time.Now().Unix(),
		}

		Verbose("dns zone create:", host, rcd)
		validHost.Store(host+".", rcd)
		http.Redirect(w, r, redirectURL, 302)
	})

	http.HandleFunc("/feedback", func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if v, ok := validHost.Load(host + "."); ok {
			rdc := v.(Record)
			rdc.HTTPRemoteAddr = r.RemoteAddr

			buf, _ := json.Marshal(rdc)
			w.Write(buf)
			validHost.Delete(host)
		} else {
			http.NotFound(w, r)
			return
		}
	})

	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		if len(r.Question) == 0 {
			return
		}

		q := r.Question[0]
		Verbose("dns query:", q.String())

		if v, ok := validHost.Load(q.Name); ok {
			rcd := v.(Record)
			raddr := parseAddr(w.RemoteAddr().String())
			rcd.DNSRemoteAddr = raddr.String()

			opt := r.IsEdns0()
			if opt != nil {
				for _, v := range opt.Option {
					switch v.(type) {
					case *dns.EDNS0_SUBNET:
						rcd.EDNSClientSubnet = v.String()
						break
					}
				}
			}

			Verbose("dns zone update:", q.Name, rcd)
			validHost.Store(q.Name, rcd)
		}

		m := new(dns.Msg)
		m.SetReply(r)

		aRec := &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    0,
			},
			A: dnsARecord,
		}
		m.Answer = append(m.Answer, aRec)
		w.WriteMsg(m)
	})
}
