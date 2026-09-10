package originhttp

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
)

type ipResolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type dialContextFunc func(context.Context, string, string) (net.Conn, error)

type safeDialer struct {
	resolver        ipResolver
	dialContext     dialContextFunc
	allowedNetworks []netip.Prefix
}

func (d *safeDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("split origin address: %w", err)
	}

	addresses, err := d.lookup(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("resolve origin host %q: no addresses", host)
	}
	for _, address := range addresses {
		if !d.isAllowed(address) {
			return nil, fmt.Errorf("origin host %q resolves to forbidden address %s", host, address)
		}
	}

	var lastErr error
	for _, address := range addresses {
		conn, err := d.dialContext(ctx, network, net.JoinHostPort(address.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("dial origin host %q: %w", host, lastErr)
}

func (d *safeDialer) lookup(ctx context.Context, host string) ([]netip.Addr, error) {
	if address, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{address.Unmap()}, nil
	}
	addresses, err := d.resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve origin host %q: %w", host, err)
	}
	for index := range addresses {
		addresses[index] = addresses[index].Unmap()
	}
	return addresses, nil
}

func (d *safeDialer) isAllowed(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || address.IsUnspecified() || address.IsMulticast() ||
		(!address.IsGlobalUnicast() && !address.IsLoopback() && !address.IsLinkLocalUnicast()) {
		return false
	}
	for _, network := range d.allowedNetworks {
		if network.Contains(address) {
			return true
		}
	}
	return isPublicAddress(address)
}

func parseAllowedCIDRs(values []string) ([]netip.Prefix, error) {
	networks := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		network, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("parse ORIGIN_ALLOWED_CIDRS value %q: %w", value, err)
		}
		networks = append(networks, network.Masked())
	}
	return networks, nil
}

func isPublicAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() {
		return false
	}
	for _, network := range forbiddenNetworks {
		if network.Contains(address) {
			return false
		}
	}
	return true
}

var forbiddenNetworks = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),    // shared address space
	netip.MustParsePrefix("168.63.129.16/32"), // Azure platform virtual address
	netip.MustParsePrefix("198.18.0.0/15"),    // benchmarking
	netip.MustParsePrefix("64:ff9b:1::/48"),   // local-use NAT64 translation
	netip.MustParsePrefix("100::/64"),         // discard-only
	netip.MustParsePrefix("2001:db8::/32"),    // documentation
	netip.MustParsePrefix("2002::/16"),        // 6to4 can encode forbidden IPv4
}
