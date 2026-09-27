package checker

import (
	"context"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// checkPing sends one ICMP echo over an unprivileged datagram socket, so no
// root or CAP_NET_RAW is needed. Linux only allows that for groups listed in
// net.ipv4.ping_group_range, which Docker opens up by default.
func (c *Checker) checkPing(ctx context.Context, host string) Result {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return failed("ping failed", err, 0)
	}
	ip := ips[0]

	network, listen, proto := "udp4", "0.0.0.0", 1
	var echo, reply icmp.Type = ipv4.ICMPTypeEcho, ipv4.ICMPTypeEchoReply
	if ip.To4() == nil {
		network, listen, proto = "udp6", "::", 58
		echo, reply = ipv6.ICMPTypeEchoRequest, ipv6.ICMPTypeEchoReply
	}

	conn, err := icmp.ListenPacket(network, listen)
	if err != nil {
		return failed("ping failed, check net.ipv4.ping_group_range", err, 0)
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	msg, err := (&icmp.Message{Type: echo, Body: &icmp.Echo{Seq: 1, Data: []byte("beacon")}}).Marshal(nil)
	if err != nil {
		return failed("ping failed", err, 0)
	}

	start := time.Now()
	if _, err := conn.WriteTo(msg, &net.UDPAddr{IP: ip}); err != nil {
		return failed("ping failed", err, 0)
	}

	// The kernel only hands this socket replies to its own echo id, but other
	// ICMP messages like unreachable can still show up.
	buf := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			return failed("ping failed", err, time.Since(start).Milliseconds())
		}
		m, err := icmp.ParseMessage(proto, buf[:n])
		if err == nil && m.Type == reply {
			return Result{IsUp: true, ResponseTime: time.Since(start).Milliseconds()}
		}
	}
}
