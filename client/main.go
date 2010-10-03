package main

import "os"
import "fmt"

func main() {
	var err os.Error

	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "e Missing arguments.\n")
		Usage()
		os.Exit(1)
	}

	clnt := NewClient()
	if err = clnt.Run(os.Args[1], os.Args[2], os.Args[3]); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}

func Usage() {
	fmt.Fprintf(os.Stdout, `Usage: %s <localip> <publicip:publicport> <destip:dstport>

  localip : Local ip address for this machine (router's DHCP address behind NAT)
 publicip : Source address to bind to and listen on. Address visible to 
            destination and/or outside world.
     dest : Destination address to send to/receive from.

IP can be in IPv4 or IPv6 format. IPv6 address should be encased in brackets.
Example: %s [fe80::222:15ff:fe65:b2f9] 62.159.23.15:54322 80.233.1.100:54321
`,
		os.Args[0], os.Args[0])
}
