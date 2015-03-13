package main

import "os"
import "net"
import "os/signal"
import "github.com/snuk182/gnarly/network"
import "bytes"
import "bufio"
import "fmt"

type Client struct {
	peer *network.Peer
}

func NewClient() *Client {
	return new(Client)
}

func (this *Client) Run(addr string) (err error) {
	// Resolve/Validate the user supplied address
	var pubaddr *net.UDPAddr
	if pubaddr, err = net.ResolveUDPAddr("udp", addr); err != nil {
		return
	}

	// Get our clientID from the local IP address
	var clientid []byte
	if clientid, err = network.GetClientId(); err != nil {
		return
	}

	// Create a new peer instance. This is our main network client. We use it
	// to listen for incoming data.
	if this.peer, err = network.NewPeer(pubaddr, clientid); err != nil {
		return
	}

	// Create a message handler. We need a closure, because we can't pass
	// a struct method as a function 'pointer'.
	mh := func(c *network.Peer, mt uint8, d interface{}) { this.onMessage(c, mt, d) }
	eh := func(err error) bool { return this.onError(err) }

	// Start the listener. 5 second ping interval and 3 minute timeout treshold.
	if err = this.peer.Listen(5e9, 180, mh, eh); err != nil {
		return
	}

	fmt.Printf("[i] Listening on: %v\n", pubaddr)

	// Hook up the input polling from stdin.
	go this.input(pubaddr)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	<-c

	fmt.Println("[i] Shutting down")
	this.Close()
	return
}

func (this *Client) Close() {
	if this.peer != nil {
		this.peer.Close()
		this.peer = nil
	}
}

func (this *Client) onMessage(peer *network.Peer, msgtype uint8, data interface{}) {
	switch msgtype {
	case network.MsgPeerConnected:
		fmt.Printf("[i] Peer connected: %s\n", peer.Id)
	case network.MsgPeerDisconnected:
		fmt.Printf("[i] Peer disconnected: %s\n", peer.Id)
	case network.MsgLatency:
		fmt.Printf("[i] Latency for %v: %d microseconds\n", peer.Id, data.(uint16))
	case network.MsgData:
		fmt.Printf("[i] From: %v\n", peer.Id)
		fmt.Printf("[i] Sequence #: 0x%04x\n", peer.Sequence)
		fmt.Printf("[i] Data: %+v\n\n", data.([]byte))
	}
}

func (this *Client) onError(err error) bool {
	fmt.Fprintf(os.Stderr, "[e] %v\n", err)
	return false
}

func (this *Client) input(addr *net.UDPAddr) {
	var line, data []byte
	var err error
	var size int

	fmt.Fprint(os.Stdout, "[i] Type some text and hit <enter> or ctrl-c to quit.\n")

	buf := bufio.NewReader(os.Stdin)
	newline := [2][]byte{[]byte{'\\', 'n'}, []byte{'\n'}}
	tab := [2][]byte{[]byte{'\\', 't'}, []byte{'\t'}}

	for {
		if line, err = buf.ReadBytes('\n'); err != nil {
			fmt.Fprintf(os.Stderr, "[e] %v\n", err)
			continue
		}

		if line = bytes.TrimSpace(line); len(line) == 0 {
			continue
		}

		line = bytes.Join(bytes.Split(line, newline[0]), newline[1])
		line = bytes.Join(bytes.Split(line, tab[0]), tab[1])

		size = len(line) + 1
		if size >= cap(data) {
			data = make([]byte, size, size)
		}

		data = data[0:size]
		data[0] = network.MsgData
		copy(data[1:], line)

		if err = this.peer.Send(addr, data); err != nil {
			fmt.Fprintf(os.Stderr, "[e] %v\n", err)
		}
	}
}
