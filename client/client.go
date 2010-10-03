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

	var clientid []byte
	var msg *network.Message
	var pubaddr *net.UDPAddr

	this.info.Log("Connecting...")

	// Get our clientID from the local IP address
	if clientid, err = network.GetClientId(local); err != nil {
		this.error.Log(err.String())
		return
	}

	// Resolve our public IP address
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

	// Start the listener. 10 second ping interval and 3 minute timeout treshold.
	if err = this.peer.Listen(1e10, 180); err != nil {
		this.error.Log(err.String())
		return
	}

	this.info.Logf("Listening on: %s", public)
	this.info.Logf("Using client Id: %s", this.peer.Id)

	errors := this.peer.Errors()
	messages := this.peer.Messages()

	// Hook up the input polling from stdin.
	go this.input(dest)

	// Main application loop. Wait for a message, error or a termination signal.
	// Ideally you should do something useful with the incoming messages. Right
	// now they are simple plain-text parroting of the commandline input. In a
	// 'real' aplication, these would be binary encoded messages with specific
	// functions.

	var client *network.Peer

loop:
	for {
		select {
		case msg = <-messages:
			if client = this.peer.GetClient(msg.PeerId); client != nil {
				this.info.Logf("Latency: %d microseconds", client.GetLatency())
			}

			switch msg.Type {
			case network.MsgPeerConnected:
				this.info.Logf("Peer connected: %s", msg.PeerId)
			case network.MsgPeerDisconnected:
				this.info.Logf("Peer disconnected: %s", msg.PeerId)
			case network.MsgData:
				this.info.Logf("Data: %v", string(msg.Data))
			}

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
	var line, data []byte
	var err os.Error
	var size int

	var addr *net.UDPAddr
	if addr, err = net.ResolveUDPAddr(dest); err != nil {
		return
	}

	this.info.Log("Type something and hit <enter>:")

	buf := bufio.NewReader(os.Stdin)
	for {
		if line, err = buf.ReadBytes('\n'); err != nil {
			this.error.Log(err.String())
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
			this.error.Log(err.String())
		}
	}
}
