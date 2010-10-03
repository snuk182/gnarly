package main

import "os"
import "log"
import "net"
import "os/signal"
import "gnarly/network"
import "bytes"
import "bufio"

type Client struct {
	info  *log.Logger
	error *log.Logger
	peer  *network.Peer
}

func NewClient() *Client {
	f := log.Ltime | log.Lmicroseconds

	c := new(Client)
	c.info = log.New(os.Stdout, nil, "i ", f)
	c.error = log.New(os.Stderr, nil, "e ", f)
	return c
}

func (this *Client) Run(local, public, dest string) (err os.Error) {
	if this.peer != nil {
		return
	}

	this.info.Log("Connecting...")

	// Get our clientID from the local IP address
	var clientid []byte
	if clientid, err = network.GetClientId(local); err != nil {
		this.error.Log(err.String())
		return
	}

	// Resolve our public IP address
	var pubaddr *net.UDPAddr
	if pubaddr, err = net.ResolveUDPAddr(public); err != nil {
		this.error.Log(err.String())
		return
	}

	// Create a new peer instance. This is our main network client. We use it
	// to listen for incoming data.
	if this.peer, err = network.NewPeer(pubaddr, clientid); err != nil {
		this.error.Log(err.String())
		return
	}

	// Start the listener
	if err = this.peer.Listen(); err != nil {
		this.error.Log(err.String())
		return
	}

	// Hook up the input polling from stdin.
	go this.input(dest)

	this.info.Logf("Listening on: %s", public)
	this.info.Logf("Using client Id: %v", this.peer.GetId())

	errors := this.peer.Errors()
	messages := this.peer.Messages()

	var msg *network.Message

	// Main application loop. Wait for a message, error or a termination signal.

loop:
	for {
		select {
		case msg = <-messages:
			this.info.Logf("[%v] -> %s", []byte(msg.PeerId), msg.Data)

		case err = <-errors:
			this.error.Log(err.String())

		case sig := <-signal.Incoming:
			if usig, ok := sig.(signal.UnixSignal); ok {
				switch usig {
				case signal.SIGINT, signal.SIGTERM, signal.SIGKILL:
					break loop
				}
			}
		}
	}

	this.info.Log("Shutting down")
	this.Close()
	return
}

func (this *Client) Close() {
	if this.peer != nil {
		this.peer.Close()
		this.peer = nil
	}
}

func (this *Client) input(dest string) {
	var line []byte
	var err os.Error

	buf := bufio.NewReader(os.Stdin)
	for {
		if line, err = buf.ReadBytes('\n'); err != nil {
			this.error.Log(err.String())
			continue
		}

		if line = bytes.TrimSpace(line); len(line) == 0 {
			continue
		}

		if err = this.peer.Send(dest, line); err != nil {
			this.error.Log(err.String())
		}
	}
}
