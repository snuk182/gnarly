package main

import "os"
import "fmt"

func main() {
	var err os.Error

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "[e] Missing arguments.\n")
		Usage()
		os.Exit(1)
	}

	clnt := NewClient()
	if err = clnt.Run(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "[e] %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func Usage() {
	fmt.Fprintf(os.Stdout, `Usage: %s <ip:port>

 ip/port : Address on which to listen and receive. This client talks to itself.

IP can be in IPv4 or IPv6 format. IPv6 address should be encased in brackets.
Examples: %s [fe80::222:15ff:fe65:b2f9]:54322
          %s 80.233.1.100:54321
`,
		os.Args[0], os.Args[0], os.Args[0])
}
