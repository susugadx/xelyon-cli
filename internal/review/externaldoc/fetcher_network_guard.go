package externaldoc

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
)

var (
	reviewExternalDocBlockedIPPrefixes = mustReviewExternalDocBlockedIPPrefixes(
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"255.255.255.255/32",
		"::/128",
		"::1/128",
		"::ffff:0:0/96",
		"64:ff9b::/96",
		"100::/64",
		"2001::/32",
		"2001:2::/48",
		"2001:db8::/32",
		"fc00::/7",
		"fe80::/10",
		"ff00::/8",
	)
)

type reviewExternalDocNetworkGuard struct {
	lookupIPAddr     func(context.Context, string) ([]net.IPAddr, error)
	dialContext      func(context.Context, string, string) (net.Conn, error)
	allowPrivateAddr bool
}

func (f *HTTPFetcher) effectiveNetworkGuard() reviewExternalDocNetworkGuard {
	if f != nil && (f.networkGuard.lookupIPAddr != nil || f.networkGuard.dialContext != nil || f.networkGuard.allowPrivateAddr) {
		guard := f.networkGuard
		guard.setDefaults()
		return guard
	}
	return newReviewExternalDocNetworkGuard()
}

func newReviewExternalDocNetworkGuard() reviewExternalDocNetworkGuard {
	guard := reviewExternalDocNetworkGuard{}
	guard.setDefaults()
	return guard
}

func (g *reviewExternalDocNetworkGuard) setDefaults() {
	if g.lookupIPAddr == nil {
		g.lookupIPAddr = net.DefaultResolver.LookupIPAddr
	}
	if g.dialContext == nil {
		dialer := &net.Dialer{}
		g.dialContext = dialer.DialContext
	}
}

func (g reviewExternalDocNetworkGuard) validateHost(ctx context.Context, host string) error {
	if g.allowPrivateAddr {
		return nil
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("external doc host is required")
	}
	_, err := g.publicIPsForHost(ctx, host)
	return err
}

func (g reviewExternalDocNetworkGuard) publicIPsForHost(ctx context.Context, host string) ([]netip.Addr, error) {
	g.setDefaults()
	if ip, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
		ip = ip.Unmap()
		if !reviewExternalDocIsPublicRoutableIP(ip) {
			return nil, fmt.Errorf("external doc host must resolve to public routable IPs")
		}
		return []netip.Addr{ip}, nil
	}

	addrs, err := g.lookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("external doc host lookup failed: %w", err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("external doc host lookup returned no addresses")
	}

	ips := make([]netip.Addr, 0, len(addrs))
	for _, resolved := range addrs {
		ip, ok := netip.AddrFromSlice(resolved.IP)
		if !ok {
			return nil, fmt.Errorf("external doc host lookup returned an invalid address")
		}
		ip = ip.Unmap()
		if !reviewExternalDocIsPublicRoutableIP(ip) {
			return nil, fmt.Errorf("external doc host must resolve to public routable IPs")
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

func (g reviewExternalDocNetworkGuard) dialContextForPublicHost(ctx context.Context, network, address string) (net.Conn, error) {
	if g.allowPrivateAddr {
		g.setDefaults()
		return g.dialContext(ctx, network, address)
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := g.publicIPsForHost(ctx, host)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, ip := range ips {
		if network == "tcp4" && !ip.Is4() {
			continue
		}
		if network == "tcp6" && !ip.Is6() {
			continue
		}
		conn, err := g.dialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("external doc host has no address for network %s", network)
}

func reviewExternalDocIsPublicRoutableIP(ip netip.Addr) bool {
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, prefix := range reviewExternalDocBlockedIPPrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func mustReviewExternalDocBlockedIPPrefixes(values ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefixes = append(prefixes, netip.MustParsePrefix(value))
	}
	return prefixes
}
