// Command vxaos-input sends test-only input to a VXAOS client.
//
// The client is started with VXAOS_TEST_INPUT set to a port (for example 5001)
// and then accepts one command per connection on 127.0.0.1. This does not use
// the physical mouse/keyboard or change the foreground window.
//
// Usage:
//
//	vxaos-input type "secret"
//	vxaos-input key enter press 80
//	vxaos-input key d press
//	vxaos-input click 476 329
//	vxaos-input move 400 300
//	vxaos-input reset
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:5001", "test-input endpoint")
	timeout := flag.Duration("timeout", 2*time.Second, "connect timeout")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	command := strings.Join(args, " ")
	conn, err := net.DialTimeout("tcp", *addr, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vxaos-input: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(*timeout))
	if _, err := fmt.Fprintln(conn, command); err != nil {
		fmt.Fprintf(os.Stderr, "vxaos-input: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: vxaos-input [-addr 127.0.0.1:5001] <command>

commands:
  key <name|vk> down|up|press [ms]   e.g. key enter press 80
  type <text>                        ASCII text, newline presses Enter
  move <x> <y>                        client coordinates
  click <x> <y> [left|right|middle]   client coordinates
  reset                              release every injected key/button`)
}
