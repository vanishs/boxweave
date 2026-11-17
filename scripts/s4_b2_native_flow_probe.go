//go:build s4b2_native_ci

// Disposable Incus CI-only long-lived TCP/UDP echo probe; never in the service.
package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

func marker(path, value string) {
	if err := os.WriteFile(path, []byte(value+"\n"), 0600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
func server() error {
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
			conn, e := tcp.Accept()
			if e != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				var data [2048]byte
				for {
					n, e := c.Read(data[:])
					if n > 0 {
						if _, err := c.Write(data[:n]); err != nil {
							return
						}
					}
					if e != nil {
						return
					}
				}
			}(conn)
		}
	}()
	marker("/tmp/s4b2-flow-server-ready", "READY")
	var b [2048]byte
	for {
		n, addr, e := udp.ReadFrom(b[:])
		if e != nil {
			return e
		}
		if _, e = udp.WriteTo(b[:n], addr); e != nil {
			return e
		}
	}
}
func client(protocol, host, label string) error {
	port := "19091"
	if protocol == "udp" {
		port = "19092"
	}
	c, err := net.DialTimeout(protocol, net.JoinHostPort(host, port), 6*time.Second)
	if err != nil {
		return fmt.Errorf("cannot establish %s: %w", label, err)
	}
	defer c.Close()
	base := "/tmp/s4b2-flow-" + label
	var reply [1]byte
	for i := 1; i <= 1000; i++ {
		if err = c.SetDeadline(time.Now().Add(1200 * time.Millisecond)); err != nil {
			return err
		}
		if _, err = c.Write([]byte{'x'}); err == nil {
			_, err = c.Read(reply[:])
		}
		if err != nil {
			marker(base+"-result", "BLOCKED_AFTER="+strconv.Itoa(i-1))
			return nil
		}
		marker(base+"-count", strconv.Itoa(i))
		if i == 1 {
			marker(base+"-ready", "READY")
		}
		time.Sleep(160 * time.Millisecond)
	}
	marker(base+"-result", "UNBLOCKED_AFTER=1000")
	return fmt.Errorf("ACL did not block %s after 1000 echo packets", label)
}
func main() {
	var err error
	switch {
	case len(os.Args) == 2 && os.Args[1] == "server":
		err = server()
	case len(os.Args) == 5 && os.Args[1] == "client" && (os.Args[2] == "tcp" || os.Args[2] == "udp"):
		err = client(os.Args[2], os.Args[3], os.Args[4])
	default:
		err = fmt.Errorf("bad disposable CI probe arguments")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
