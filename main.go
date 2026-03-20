package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	server := flag.String("server", "8.8.8.8:53", "DNS server address, e.g. 8.8.8.8:53")
	name := flag.String("name", "", "Domain or value to query, e.g. example.com")
	qtype := flag.String("type", "A", "Record type: A, AAAA, CNAME, MX, NS, TXT, PTR")
	transport := flag.String("transport", "udp", "Transport: udp or tcp")
	timeout := flag.Duration("timeout", 3*time.Second, "Request timeout, e.g. 5s")
	flag.Parse()

	if *name == "" {
		fmt.Fprintln(os.Stderr, "missing required flag: -name")
		flag.Usage()
		os.Exit(1)
	}

	serverAddr, err := normalizeServerAddr(*server)
	if err != nil {
		exitWithError(err)
	}

	transportName := strings.ToLower(strings.TrimSpace(*transport))
	if transportName != "udp" && transportName != "tcp" {
		exitWithError(fmt.Errorf("invalid -transport %q, must be udp or tcp", *transport))
	}

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: *timeout}
			return dialer.DialContext(ctx, transportName, serverAddr)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if err := query(ctx, resolver, strings.TrimSpace(*name), strings.ToUpper(strings.TrimSpace(*qtype))); err != nil {
		exitWithError(err)
	}
}

func query(ctx context.Context, resolver *net.Resolver, name, qtype string) error {
	switch qtype {
	case "A":
		ips, err := resolver.LookupIP(ctx, "ip4", name)
		if err != nil {
			return err
		}
		for _, ip := range ips {
			fmt.Println(ip.String())
		}
	case "AAAA":
		ips, err := resolver.LookupIP(ctx, "ip6", name)
		if err != nil {
			return err
		}
		for _, ip := range ips {
			fmt.Println(ip.String())
		}
	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, name)
		if err != nil {
			return err
		}
		fmt.Println(cname)
	case "MX":
		records, err := resolver.LookupMX(ctx, name)
		if err != nil {
			return err
		}
		for _, r := range records {
			fmt.Printf("%s %d\n", r.Host, r.Pref)
		}
	case "NS":
		records, err := resolver.LookupNS(ctx, name)
		if err != nil {
			return err
		}
		for _, r := range records {
			fmt.Println(r.Host)
		}
	case "TXT":
		records, err := resolver.LookupTXT(ctx, name)
		if err != nil {
			return err
		}
		for _, r := range records {
			fmt.Println(r)
		}
	case "PTR":
		records, err := resolver.LookupAddr(ctx, name)
		if err != nil {
			return err
		}
		for _, r := range records {
			fmt.Println(r)
		}
	default:
		return fmt.Errorf("unsupported -type %q, supported: A, AAAA, CNAME, MX, NS, TXT, PTR", qtype)
	}

	return nil
}

func normalizeServerAddr(raw string) (string, error) {
	addr := strings.TrimSpace(raw)
	if addr == "" {
		return "", fmt.Errorf("server address is empty")
	}

	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr, nil
	}

	if strings.Count(addr, ":") > 1 {
		// IPv6 without port.
		return net.JoinHostPort(addr, "53"), nil
	}

	return net.JoinHostPort(addr, "53"), nil
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
