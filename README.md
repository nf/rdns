# rdns

`rdns` provides reverse DNS for IPv6 networks. It is a DNS server that
serves PTR and AAAA records for an IPv6 subnet, and the corresponding
authoritative NS records.

## Setup

- Run `rdns` somewhere serving on port `53` (set flags appropriately).
- Ask your upstream IPv6 provider to create an `ip6.arpa.` NS record for your
  subnet that points to your `rdns` instance.
- Create an NS record for your subdomain (eg, `v6.example.com`) that also
  points to your `rdns` instance.

### Command-line flags

`rdns` is configured with command-line flags:

```
  -domain string
        domain suffix for generated domain name (default ".v6.example.com.")
  -host_prefix string
        prefix for generated host name (default "ip-")
  -listen address
        DNS server listen address (default ":53")
  -network subnet
        subnet for which to serve ip6.arpa records (default "fe80::/64")
  -ns server
        name server for NS responses (default "ns.example.com.")
  -ttl seconds
        answer TTL in seconds (default 3600)
```

## Demo

Here are a few `dig` queries against `rdns` running with its default flags:

```
$ dig -x fe80::f00d

;; QUESTION SECTION:
;d.0.0.f.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.e.f.ip6.arpa. IN PTR

;; ANSWER SECTION:
d.0.0.f.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.e.f.ip6.arpa. 3600 IN PTR ip-f00d.v6.example.com.


$ dig -t AAAA ip-f00d.v6.example.com

;; QUESTION SECTION:
;ip-f00d.v6.example.com.                IN      AAAA

;; ANSWER SECTION:
ip-f00d.v6.example.com. 3600    IN      AAAA    fe80::f00d


$ dig -t NS 0.0.0.0.0.0.0.0.0.0.0.0.0.8.e.f.ip6.arpa.

;; QUESTION SECTION:
;0.0.0.0.0.0.0.0.0.0.0.0.0.8.e.f.ip6.arpa. IN NS

;; AUTHORITY SECTION:
0.0.0.0.0.0.0.0.0.0.0.0.0.8.e.f.ip6.arpa. 3600 IN NS ns.example.com.
```

## Acknowledgements

I made this because I wanted to serve reverse DNS for my IPv6 subnet. The first
option I tried was [AllKnowingDNS](https://all-knowing-dns.zekjur.net/), for
which there is a package on my Debian system, but it failed with a Perl runtime
error due to a missing method on some object. Rather than try to debug Perl
code I took the opportunity to learn more about DNS and write my own.
It was a very informative experience, and was quite straightforward thanks to
Miek Gieben's excellent [dns package](https://github.com/miekg/dns).

## Copyright and License

This code is Copyright 2022 Google Inc. and is [Apache 2.0](LICENSE) licensed.
