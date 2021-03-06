package network

import (
	"bytes"
	"encoding/base64"
	"net"
	"sync"
	"time"
)

// This type represents a function handler for dealing with incoming messages.
type MessageHandler func(client *Peer, msgtype uint8, data interface{})

// This type represents a function handler for dealing with error messages.
type ErrorHandler func(err error) bool

// This represents a unique client connecting to our machine. This structure
// maintains some counters and buffers used for reliable identification and
// caching of the data packets sent to/from said client.
type Peer struct {
	Id          string       // 24 byte base64 encoded Md5 hash identifying this peer.
	clientId    []uint8      // 2 byte client id.
	Addr        *net.UDPAddr // Public address for this peer.
	Sequence    uint16       // This counter keeps track of the amount of packets we sent to the receiver.
	latencydata [2]uint32    // Total Packet count and Total Packet rountrip time in microseconds for each PING request.
	lastpacket  int64        // Last packet receive time. Used for timeout detection.
	scratch     []uint8      // A temporary data buffer.

	// Fields only used by a listening peer.
	onMessage MessageHandler   // function pointer to a message handler.
	onError   ErrorHandler     // function pointer to error handler
	udp       *net.UDPConn     // Our UDP listener socket.
	clients   map[string]*Peer // List of known clients we rceived data from in this session.
	cache     []Packet         // Cache of packets. Used when expecting a sequence.
	ticker    *time.Ticker     // Used for ping requests when this peer is functioning as a listener.
	lock      *sync.Mutex      // Used to synchronise access to some peer fields.
	timeout   uint16           // Number of seconds a client can remain unresponsive before we consider it 'disconnected'.
}

// Constructs a new Peer instance
func NewPeer(addr *net.UDPAddr, clientid []uint8) (p *Peer, err error) {
	if len(clientid) != 2 {
		return nil, ErrInvalidClientID
	}

	p = new(Peer)
	p.clientId = clientid
	p.Addr = addr

	var d []uint8
	buf := bytes.NewBuffer(d)
	buf.Write(addr.IP.To16())
	buf.Write(clientid)

	hash := md5hash(buf.Bytes())

	buf.Truncate(0)
	enc := base64.NewEncoder(base64.StdEncoding, buf)
	if _, err = enc.Write(hash); err != nil {
		enc.Close()
		return nil, ErrInvalidClientID
	}
	enc.Close()
	p.Id = buf.String()
	return
}

// Begin listening on the public IP/port. Specify the interval at which you
// want to 'ping' known clients. This value is the number of nanoseconds you
// want between each ping. It is used to measure latency and to detect timeouts.
// The timeout argument is the number of seconds we should allow a peer to
// remain inactive before we consider it 'disconnected'.
func (this *Peer) Listen(pinginterval uint64, timeout uint16, mh MessageHandler, eh ErrorHandler) (err error) {
	if this.udp != nil {
		return
	}

	if mh == nil {
		return ErrInvalidMessageHandler
	}

	if eh == nil {
		return ErrInvalidErrorHandler
	}

	if pinginterval == 0 {
		pinginterval = 1e10 // Every 10 seconds = default
	}

	this.lock = new(sync.Mutex)
	this.lock.Lock()

	if cap(this.scratch) == 0 {
		this.scratch = make([]uint8, PacketSize-UdpHeaderSize)
	}

	this.onMessage = mh
	this.onError = eh
	this.clients = make(map[string]*Peer)
	this.ticker = time.NewTicker(time.Duration(pinginterval))
	this.timeout = timeout
	this.lock.Unlock()

	if this.udp, err = net.ListenUDP("udp", this.Addr); err != nil {
		return
	}

	go this.poll()
	go this.ping()
	return
}

// Ping every known client. We send a 64 bit timestamp. This is the current time
// in microseconds. This kind of precision adds some packet size overhead as
// opposed to a regular 4 byte unix timestamp, but we get better timing info.
// These packets are also only sent at low intervals, so the extra 4 bytes are
// not going to be a problem. We could use milliseconds, but that would still
// require a 64 bit integer. So the extra precision of microseconds adds no
// extra cost.
func (this *Peer) ping() {
	var ms int64
	var id string

	limit := int64(this.timeout) * 1e9
	data := make([]uint8, 9)
	data[0] = MsgPing

	for {
		select {
		case _ = <-this.ticker.C:
			for id = range this.clients {
				// Use this opportunity to make sure client has not timed out.
				if time.Now().UnixNano()-this.clients[id].lastpacket > limit {
					// This one has exceeded the non-response time limit. Consider it a lost cause.
					this.onMessage(this.clients[id], MsgPeerDisconnected, nil)

					this.lock.Lock()
					delete(this.clients, id)
					this.lock.Unlock()
					continue
				}

				// Send current time in microseconds to client.
				ms = time.Now().UnixNano() / 1e3

				data[1] = uint8(ms >> 56)
				data[2] = uint8(ms >> 48)
				data[3] = uint8(ms >> 40)
				data[4] = uint8(ms >> 32)
				data[5] = uint8(ms >> 24)
				data[6] = uint8(ms >> 16)
				data[7] = uint8(ms >> 8)
				data[8] = uint8(ms)
				this.send(this.clients[id].Addr, data, MsgPing)
			}
		}
	}
}

func (this *Peer) LocalAddr() net.Addr {
	if this.udp != nil {
		return this.udp.LocalAddr()
	} else {
		return nil
	}
}

// Poll for incoming data
func (this *Peer) poll() {
	var err error
	var size int
	var addr *net.UDPAddr
	var stamp int64
	var i int

	datasize := PacketSize - 6 // = PacketSize-UdpHeader+len(ipv6(addr))
	data := make([]uint8, datasize, datasize)

loop:
	for this.udp != nil {
		size, addr, err = this.udp.ReadFromUDP(data[16:]) // leave room for 16-byte address
		stamp = time.Now().UnixNano()

		switch {
		case err != nil:
			if this.onError(err) {
				break loop
			}
		case size < 6: // Need 5 byte msg header + at least 1 byte data (msg id)
			if this.onError(ErrInvalidPacket) {
				break loop
			}
		default:
			for i = 0; i < 16; i++ {
				data[i] = addr.IP.To16()[i]
			}
			this.process(addr, data[0:size+16], stamp)
		}
	}
}

func (this *Peer) process(addr *net.UDPAddr, packet Packet, stamp int64) {
	var client *Peer
	var ok bool
	var data []uint8

	id := packet.Owner()

	// Create or update peer (owner of packet).
	if client, ok = this.clients[id]; !ok {
		client, _ = NewPeer(addr, packet.ClientId())

		this.lock.Lock()
		this.clients[id] = client
		this.lock.Unlock()

		this.onMessage(this.clients[id], MsgPeerConnected, nil)
	}

	this.lock.Lock()
	client.Addr = addr
	client.Sequence = packet.Sequence()
	client.lastpacket = stamp
	this.lock.Unlock()

	if len(packet) > 22 {
		if packet[18]&PFFragmented != 0 {
			// This packet is part of a sequence. We need to store it and
			// make sure we get all of them. We can then reassemble the
			// original dataset.

			s1, s2 := packet.SubSequence()
			this.lock.Lock()
			if int(s2) > len(this.cache) {
				this.cache = make([]Packet, s2)
			}
			this.cache[s1] = packet
			this.lock.Unlock()

			// Check if we have all of them
			var i int
			for i = range this.cache {
				if this.cache[i] == nil {
					return // Not yet. Stop processing
				}
			}

			// We have all members of the sequence. Reassemble it.
			buf := bytes.NewBuffer(data)

			this.lock.Lock()
			for i = range this.cache {
				buf.Write(this.cache[i].Data())
				this.cache[i] = nil
			}

			this.cache = this.cache[0:0]
			this.lock.Unlock()

			data = buf.Bytes()
		} else {
			data = packet.Data()
		}

		// Decrypt if necessary.
		if packet[18]&PFEncrypted != 0 && Encryption != nil {
			data = Encryption.Decrypt(id, data)
		}

		// Decompress if necessary.
		if packet[18]&PFCompressed != 0 && Compression != nil {
			data = Compression.Decompress(data)
		}

		// Check if we got a packet used by this lib internally (eg: ping).
		// These don't have to be forwarded to the host app.
		if len(data) > 0 {
			switch data[0] {
			case MsgPing: // respond with supplied timestamp
				data[0] = MsgPong
				this.send(addr, data[1:], MsgPong)
				return

			case MsgPong: // Calculate latency from packet rounttrip time.
				cms := time.Now().UnixNano() / 1e3
				oms := int64(data[1])<<56 | int64(data[2])<<48 | int64(data[3])<<40 |
					int64(data[4])<<32 | int64(data[5])<<24 | int64(data[6])<<16 |
					int64(data[7])<<8 | int64(data[8])

				// We average the latency out over the last 10 ping requests.
				this.lock.Lock()
				if client.latencydata[0] >= 10 {
					client.latencydata[0] = 0
					client.latencydata[1] = 0
				}

				client.latencydata[0]++
				client.latencydata[1] += uint32(cms - oms)
				this.lock.Unlock()

				this.onMessage(client, MsgLatency, uint16(client.latencydata[1]/client.latencydata[0]))
			default:
				this.onMessage(client, data[0], data[1:])
			}
		} else {
			return //ErrNoData
		}
	} else {
		return //ErrInvalidPacket
	}
}

// Close the listener
func (this *Peer) Close() {
	this.lock.Lock()

	if this.ticker != nil {
		this.ticker.Stop()
		this.ticker = nil
	}

	if this.udp != nil {
		this.udp.Close()
		this.udp = nil
	}

	// Give code some time to break out of polling loop
	time.Sleep(1e9)

	this.lock.Unlock()
	this.lock = nil
}

// This sends the given data to the given address. It takes care of building
// the packets with accurate header information. If the length of the supplied
// data exceeds the established packet size (minus the UDP + message headers),
// it will also take care of the required packet fragmentation so all the
// information is sent. If network.Compressed and/or network.Encrypted are set.
// this will also make sure these operations are performed on the data.
func (this *Peer) Send(addr *net.UDPAddr, data []uint8) (err error) {
	return this.send(addr, data, MsgData)
}

func (this *Peer) send(addr *net.UDPAddr, data []uint8, msgtype uint8) (err error) {
	this.lock.Lock()
	this.lock.Unlock()

	if cap(this.scratch) == 0 {
		this.scratch = make([]uint8, PacketSize-UdpHeaderSize)
	}

	this.scratch[0] = this.clientId[0]
	this.scratch[1] = this.clientId[1]
	this.scratch[2] = 0

	if Compression != nil {
		data = Compression.Compress(data)
		this.scratch[2] |= PFCompressed
	}

	if Encryption != nil {
		data = Encryption.Encrypt(this.Id, data)
		this.scratch[2] |= PFEncrypted
	}

	if len(data) > PacketSize-UdpHeaderSize-5 {
		// Packet fragmentation required because data exceeds available packet space.
		this.scratch[2] |= PFFragmented
		size := PacketSize - UdpHeaderSize - 8

		var cur, total uint8
		total = uint8(len(data) / size)

		if len(data)%size > 0 {
			total++
		}

		// Build and send as many packets as needed.
		for {
			// FIXME(jimt): Handle wrapping of this.Sequence value if it exceeds uint16

			this.scratch[3] = uint8(this.Sequence >> 8)
			this.scratch[4] = uint8(this.Sequence)
			if this.Sequence == 65535 { //attempt at seqeucnce wrap
				this.Sequence = 0
			} else {
				this.Sequence++
			}

			this.scratch[5] = cur
			this.scratch[6] = total
			this.scratch[7] = msgtype
			cur++

			copy(this.scratch[8:], data)
			if err = this.sendToSocket(addr, this.scratch[0:size+8]); err != nil {
				return
			}
			data = data[size:]

			if len(data) <= size {
				break
			}
		}

		// Send any remaining data
		if len(data) > 0 {
			// FIXME(jimt): Handle wrapping of this.Sequence value if it exceeds uint16
			this.scratch[3] = uint8(this.Sequence >> 8)
			this.scratch[4] = uint8(this.Sequence)
			if this.Sequence == 65535 { //attempt at seqeucnce wrap
				this.Sequence = 0
			} else {
				this.Sequence++
			}

			this.scratch[5] = cur
			this.scratch[6] = total
			this.scratch[7] = msgtype
			copy(this.scratch[8:], data)

			return this.sendToSocket(addr, this.scratch[0:len(data)+8])
		}

	} else {
		// Single packet. Just send as-is

		// FIXME(jimt): Handle wrapping of this.Sequence value if it exceeds uint16
		this.scratch[3] = uint8(this.Sequence >> 8)
		this.scratch[4] = uint8(this.Sequence)
		this.scratch[5] = msgtype
			if this.Sequence == 65535 { //attempt at seqeucnce wrap
			this.Sequence = 0
		} else {
			this.Sequence++
		}
		copy(this.scratch[6:], data)
		return this.sendToSocket(addr, this.scratch[0:len(data)+6])
	}
	return
}

// Called from Peer.Send()
func (this *Peer) sendToSocket(addr *net.UDPAddr, data []uint8) (err error) {
	if this.udp != nil {
		// If this is a listening peer, just reuse the existing connection for sending.
		_, err = this.udp.WriteToUDP(data, addr)
	} else {
		// Otherwise, create a new one.
		var conn *net.UDPConn

		if conn, err = net.DialUDP("udp", nil, addr); err != nil {
			return
		}

		defer conn.Close()
		_, err = conn.WriteToUDP(data, addr)
	}

	return
}

// Finds the known peer with the given ID
func (this *Peer) GetClient(id string) *Peer {
	if p, ok := this.clients[id]; ok {
		return p
	}
	return nil
}

// Check to see if the given clientid is still listed.
func (this *Peer) HasClient(id string) bool {
	_, ok := this.clients[id]
	return ok
}

// Adds a new peer to the list of known peers
func (this *Peer) AddClient(p *Peer) {
	if _, ok := this.clients[p.Id]; ok {
		return
	}

	this.lock.Lock()
	this.clients[p.Id] = p
	this.lock.Unlock()
}

// Removes the known peer with the given id
func (this *Peer) RemoveClient(id string) {
	this.lock.Lock()
	delete(this.clients, id)
	this.lock.Unlock()
}
