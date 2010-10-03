package network

import "os"
import "net"
import "bytes"
import "crypto/md5"

// This represents a unique client connecting to our machine. This structure
// maintains some counters and buffers used for reliable identification and
// caching of the data packets sent to/from said client.
type Peer struct {
	PacketCount uint16           // This counter keeps track of the amount of packets we sent to the receiver.
	Addr        *net.UDPAddr     // Public address for this peer
	clientId    []byte           // 2 byte client id
	scratch     []byte           // A temporary data buffer
	conn        *udpListener     // UDP listener
	clients     map[string]*Peer // list of known clients we rceived data from in this session
	messages    chan *Message    // Outgoing messages with received data structures.
	cache       []Packet         // cache of datagrams. uses when expecting a sequence
}

// Constructs a new Peer instance
func NewPeer(addr *net.UDPAddr, clientid []byte) (p *Peer, err os.Error) {
	if len(clientid) != 2 {
		return nil, ErrInvalidClientID
	}

	p = new(Peer)
	p.clientId = clientid
	p.Addr = addr
	return
}

// Begin listening on the public IP/port
func (this *Peer) Listen() (err os.Error) {
	if this.conn != nil {
		return
	}

	if cap(this.scratch) == 0 {
		this.scratch = make([]byte, PacketSize-UdpHeaderSize)
	}

	this.clients = make(map[string]*Peer)
	this.conn = newUdpListener()
	this.messages = make(chan *Message)

	if err = this.conn.Run(this.Addr); err != nil {
		return
	}

	go this.poll()
	return
}

func (this *Peer) poll() {
	var dg *datagram
	var client *Peer
	var ok bool
	var id string
	var s1, s2 uint8
	var err os.Error
	var d []byte
	var i int

	buf := bytes.NewBuffer(d)

loop:
	for this.conn != nil {
		select {
		// case err = <-this.conn.errors: // Should be processed by host app
		case dg = <-this.conn.in:
			id = string(dg.Packet.Owner())

			// Create or update peer (owner of packet).
			if client, ok = this.clients[id]; !ok {
				client, _ = NewPeer(dg.Addr, dg.Packet.ClientId())
				this.clients[id] = client
			}

			// We have to keep updating the address. Some clients switch
			// ports at random. We need the latest up-to-date IP+port so our
			// outgoing data arrives at the right place.
			client.Addr = dg.Addr

			if dg.Packet[18]&PFFragmented != 0 {
				// This packet is part of a sequence. We need to store it and
				// make sure we get all of them. We can then reassemble the
				// original datastructure.

				// TODO(jimt): We should possible check if the current packet is
				// part if the same sequence as the one we have in cache. We can
				// do this by comparing the first received sequence number
				// against the current packet's first subsequence number.

				s1, s2 = dg.Packet.SubSequence()
				if int(s2) > len(this.cache) {
					this.cache = make([]Packet, s2)
				}
				this.cache[s1] = dg.Packet

				// Check if we have all of them
				for i = range this.cache {
					if this.cache[i] == nil {
						continue loop
					}
				}

				buf.Truncate(0)
				for i = range this.cache {
					buf.Write(this.cache[i].Data())
					this.cache[i] = nil
				}

				// reset cache
				this.cache = this.cache[0:0]

				if err = this.process(id, dg.Packet.Flags(), buf.Bytes()); err != nil {
					this.conn.errors <- err
				}
			} else {
				// Single message. Process it.
				if err = this.process(id, dg.Packet.Flags(), dg.Packet.Data()); err != nil {
					this.conn.errors <- err
				}
			}
		}
	}
}

func (this *Peer) process(id string, flags uint8, data []byte) (err os.Error) {
	// Decrypt if necessary.
	if flags&PFEncrypted != 0 && Decrypt != nil {
		if data, err = Decrypt([]byte(id), data); err == nil {
			return
		}
	}

	// Decompress if necessary.
	if flags&PFCompressed != 0 && Decompress != nil {
		data = Decompress(data)
	}

	// Send message with data and peerid. We don't send the whole
	// peer structure. It can be queried with Peer.GetClient() using
	// the id in the message. Most of the time the id is all the
	// host application needs to identify the sender.
	this.messages <- NewMessage(id, data)
	return
}

// Close the listener
func (this *Peer) Close() {
	if this.conn != nil {
		this.conn.Close()
		this.conn = nil
	}
}

// Returns a channel which will push any packet processing errors
func (this *Peer) Errors() <-chan os.Error { return this.conn.errors }

// Contains incoming messages from unique peers
func (this *Peer) Messages() <-chan *Message { return this.messages }

// Returns a 16 byte id identifying the unique user.
func (this *Peer) GetId() []byte {
	var d []byte

	buf := bytes.NewBuffer(d)
	buf.Write(this.Addr.IP.To16())
	buf.Write(this.clientId)

	hash := md5.New()
	hash.Write(buf.Bytes())
	return hash.Sum()
}

// This sends the given data to the given address. It takes care of building
// the packets with accurate header information. If the length of the supplied
// data exceeds the established packet size (minus the UDP + message headers),
// it will also take care of the required packet fragmentation so all the 
// information is sent. If network.Compressed and/or network.Encrypted are set.
// this will also make sure these operations are performed on the data.
func (this *Peer) Send(addr string, data []byte) (err os.Error) {
	var udpaddr *net.UDPAddr
	if udpaddr, err = net.ResolveUDPAddr(addr); err != nil {
		return
	}

	if cap(this.scratch) == 0 {
		this.scratch = make([]byte, PacketSize-UdpHeaderSize)
	}

	this.scratch[0] = this.clientId[0]
	this.scratch[1] = this.clientId[1]
	this.scratch[2] = 0

	if Compressed && Compress != nil {
		data = Compress(data)
		this.scratch[2] |= PFCompressed
	}

	if Encrypted && Encrypt != nil {
		if data, err = Encrypt(this.GetId(), data); err != nil {
			return
		}
		this.scratch[2] |= PFEncrypted
	}

	if len(data) > PacketSize-UdpHeaderSize-5 {
		// Packet fragmentation required because data exceeds available packet space.
		this.scratch[2] |= PFFragmented
		size := PacketSize - UdpHeaderSize - 7

		var cur, total uint8
		total = uint8(len(data) / size)

		if len(data)%size > 0 {
			total++
		}

		// Build and send as many packets as needed.
		for {
			// FIXME(jimt): Handle wrapping of this.PacketCount value if it exceeds uint16
			this.scratch[3] = uint8(this.PacketCount >> 8)
			this.scratch[4] = uint8(this.PacketCount)
			this.PacketCount++

			this.scratch[5] = cur
			this.scratch[6] = total
			cur++

			copy(this.scratch[7:], data)
			this.conn.out <- newDatagram(udpaddr, this.scratch[0:size+7])
			data = data[size:]

			if len(data) <= size {
				break
			}
		}

		// Send any remaining data
		if len(data) > 0 {
			// FIXME(jimt): Handle wrapping of this.PacketCount value if it exceeds uint16
			this.scratch[3] = uint8(this.PacketCount >> 8)
			this.scratch[4] = uint8(this.PacketCount)
			this.PacketCount++

			this.scratch[5] = cur
			this.scratch[6] = total
			copy(this.scratch[7:], data)
			this.conn.out <- newDatagram(udpaddr, this.scratch[0:len(data)+7])
		}

	} else {
		// Single packet. Just send as-is

		// FIXME(jimt): Handle wrapping of this.PacketCount value if it exceeds uint16
		this.scratch[3] = uint8(this.PacketCount >> 8)
		this.scratch[4] = uint8(this.PacketCount)
		this.PacketCount++

		copy(this.scratch[5:], data)
		this.conn.out <- newDatagram(udpaddr, this.scratch[0:len(data)+5])
	}
	return
}

// Finds the known peer with the given ID
func (this *Peer) GetClient(id []byte) *Peer {
	if p, ok := this.clients[string(id)]; ok {
		return p
	}
	return nil
}

// Adds a new peer to the list of known peers
func (this *Peer) AddClient(p *Peer) {
	id := string(p.GetId())
	if _, ok := this.clients[id]; ok {
		return
	}
	this.clients[id] = p
}

// Removes the known peer with the given id
func (this *Peer) RemoveClient(id []byte) {
	this.clients[string(id)] = nil, false
}
