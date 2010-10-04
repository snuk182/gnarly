package main

import "os"
import "net"
import "os/signal"
import "gnarly/network"
import "bytes"
import "bufio"
import "fmt"

type Client struct {
	peer *network.Peer
}

func NewClient() *Client {
	return new(Client)
}

func (this *Client) Run(addr string) (err os.Error) {
	// Get our clientID from the local IP address
	var clientid []byte
	if clientid, err = network.GetClientId(); err != nil {
		return
	}

	// Resolve our public IP address
	var pubaddr *net.UDPAddr
	if pubaddr, err = net.ResolveUDPAddr(addr); err != nil {
		return
	}

	// Create a new peer instance. This is our main network client. We use it
	// to listen for incoming data.
	if this.peer, err = network.NewPeer(pubaddr, clientid); err != nil {
		return
	}

	// Create a message handler. We need a closure, because we can't pass
	// a struct method as a function 'pointer'.
	handler := func(c *network.Peer, mt uint8, d []byte) { this.onMessage(c, mt, d) }

	// Start the listener. 5 second ping interval and 3 minute timeout treshold.
	if err = this.peer.Listen(5e9, 180, handler); err != nil {
		return
	}

	fmt.Fprintf(os.Stdout, "[i] Listening on: %v\n", pubaddr)

	// Hook up the input polling from stdin.
	go this.input(pubaddr)

loop:
	for {
		select {
		case sig := <-signal.Incoming:
			if usig, ok := sig.(signal.UnixSignal); ok {
				switch usig {
				case signal.SIGINT, signal.SIGTERM, signal.SIGKILL:
					break loop
				}
			}
		}
	}

	fmt.Fprint(os.Stdout, "[i] Shutting down\n")
	this.Close()
	return
}

func (this *Client) Close() {
	if this.peer != nil {
		this.peer.Close()
		this.peer = nil
	}
}

func (this *Client) onMessage(client *network.Peer, msgtype uint8, data []byte) {
	//this.info.Logf("Latency: %d microseconds", client.GetLatency())

	switch msgtype {
	case network.MsgPeerConnected:
		fmt.Fprintf(os.Stdout, "[i] Peer connected: %v\n", []byte(client.Id))
	case network.MsgPeerDisconnected:
		fmt.Fprintf(os.Stdout, "[i] Peer disconnected: %v\n", []byte(client.Id))
	case network.MsgData:
		fmt.Fprintf(os.Stdout, "[i] Data: %v\n", string(data))
	}
}

func (this *Client) input(addr *net.UDPAddr) {
	var line, data []byte
	var err os.Error
	var size int

	fmt.Fprint(os.Stdout, "[i] Type some text and hit <enter> or ctrl-c to quit.\n")

	buf := bufio.NewReader(os.Stdin)
	for {
		if line, err = buf.ReadBytes('\n'); err != nil {
			fmt.Fprintf(os.Stderr, "[e] %v\n", err)
			continue
		}

		if line = bytes.TrimSpace(line); len(line) == 0 {
			continue
		}

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
