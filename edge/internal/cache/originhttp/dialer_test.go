package originhttp

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"testing"
)

type resolverStub struct {
	addresses []netip.Addr
}

func (r resolverStub) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return r.addresses, nil
}

func TestSafeDialerRejectsForbiddenResolvedAddress(t *testing.T) {
	tests := []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.100.100.200", "::1", "fd00:ec2::254"}
	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			dialed := false
			dialer := safeDialer{
				resolver: resolverStub{addresses: []netip.Addr{netip.MustParseAddr(value)}},
				dialContext: func(context.Context, string, string) (net.Conn, error) {
					dialed = true
					return nil, errors.New("unexpected dial")
				},
			}
			_, err := dialer.DialContext(context.Background(), "tcp", "origin.example:80")
			if err == nil || !strings.Contains(err.Error(), "forbidden address") {
				t.Fatalf("DialContext() error = %v", err)
			}
			if dialed {
				t.Fatal("dialed a forbidden destination")
			}
		})
	}
}

func TestSafeDialerRejectsHostWithMixedPublicAndForbiddenAddresses(t *testing.T) {
	dialed := false
	dialer := safeDialer{
		resolver: resolverStub{addresses: []netip.Addr{
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("10.0.0.1"),
		}},
		dialContext: func(context.Context, string, string) (net.Conn, error) {
			dialed = true
			return nil, errors.New("unexpected dial")
		},
	}
	_, err := dialer.DialContext(context.Background(), "tcp", "origin.example:80")
	if err == nil || !strings.Contains(err.Error(), "forbidden address") {
		t.Fatalf("DialContext() error = %v", err)
	}
	if dialed {
		t.Fatal("dialed a host with a forbidden DNS result")
	}
}

func TestSafeDialerPinsConnectionToValidatedAddress(t *testing.T) {
	const publicIP = "93.184.216.34"
	var dialedAddress string
	dialer := safeDialer{
		resolver: resolverStub{addresses: []netip.Addr{netip.MustParseAddr(publicIP)}},
		dialContext: func(_ context.Context, _, address string) (net.Conn, error) {
			dialedAddress = address
			return nil, errors.New("stop after recording address")
		},
	}
	_, _ = dialer.DialContext(context.Background(), "tcp", "origin.example:443")
	if dialedAddress != publicIP+":443" {
		t.Fatalf("dialed address = %q", dialedAddress)
	}
}

func TestSafeDialerAllowsExplicitCIDRException(t *testing.T) {
	networks, err := parseAllowedCIDRs([]string{"10.20.30.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	var dialedAddress string
	dialer := safeDialer{
		resolver:        resolverStub{addresses: []netip.Addr{netip.MustParseAddr("10.20.30.4")}},
		allowedNetworks: networks,
		dialContext: func(_ context.Context, _, address string) (net.Conn, error) {
			dialedAddress = address
			return nil, errors.New("stop after recording address")
		},
	}
	_, _ = dialer.DialContext(context.Background(), "tcp", "internal-origin.example:80")
	if dialedAddress != "10.20.30.4:80" {
		t.Fatalf("dialed address = %q", dialedAddress)
	}
}

func TestParseAllowedCIDRsRejectsInvalidValue(t *testing.T) {
	if _, err := parseAllowedCIDRs([]string{"not-a-network"}); err == nil {
		t.Fatal("expected invalid CIDR error")
	}
}
