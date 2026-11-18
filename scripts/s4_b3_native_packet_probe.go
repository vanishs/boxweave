//go:build s4b3_native_ci

// Disposable Incus CI-only IPv4/IPv6 TCP+UDP echo assertion. Not part of the
// shipped BoxWeave binary, native helper or MCP interface.
package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func serve() error {
	tcp, err := net.Listen("tcp", "[::]:19091")
	if err != nil {
		return err
	}
	defer tcp.Close()
	udp, err := net.ListenPacket("udp", "[::]:19092")
	if err != nil {
		return err
	}
	defer udp.Close()
	go func() {
		for {
			c, e := tcp.Accept()
			if e != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				buf := make([]byte, 32)
				n, e := conn.Read(buf)
				if e == nil && n > 0 {
					_, _ = conn.Write(buf[:n])
				}
			}(c)
		}
	}()
	if err := os.WriteFile("/tmp/s4b3-packet-ready", []byte("READY\n"), 0600); err != nil {
		return err
	}
	buf := make([]byte, 64)
	for {
		n, addr, err := udp.ReadFrom(buf)
		if err != nil {
			return err
		}
		_, _ = udp.WriteTo(buf[:n], addr)
	}
}
func probe(protocol, dest string) error {
	port := "19091"
	if protocol == "udp" {
		port = "19092"
	}
	c, err := net.DialTimeout(protocol, net.JoinHostPort(dest, port), 2*time.Second)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.SetDeadline(time.Now().Add(2500 * time.Millisecond)); err != nil {
		return err
	}
	if _, err := c.Write([]byte("B3")); err != nil {
		return err
	}
	reply := make([]byte, 2)
	n, err := c.Read(reply)
	if err != nil {
		return err
	}
	if n != 2 || string(reply) != "B3" {
		return fmt.Errorf("unexpected echo %q", reply[:n])
	}
	return nil
}
func main() {
	var err error
	switch {
	case len(os.Args) == 2 && os.Args[1] == "server":
		err = serve()
	case len(os.Args) == 4 && os.Args[1] == "probe" && (os.Args[2] == "tcp" || os.Args[2] == "udp"):
		err = probe(os.Args[2], os.Args[3])
	default:
		err = fmt.Errorf("invalid disposable B3 packet probe invocation")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
