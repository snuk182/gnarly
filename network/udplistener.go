package network

import "net"
import "os"
import "sync"
import "time"

// This represents a single UDP datagram
type datagram struct {
	Addr      *net.UDPAddr
	Packet    Packet
	Timestamp int64
}

func newDatagram(addr *net.UDPAddr, data []uint8, stamp int64) *datagram {
	d := new(datagram)
	d.Timestamp = stamp
	d.Addr = addr
	d.Packet = make([]uint8, len(data))
	copy(d.Packet, data)
	return d
}

// This represents the main UDP listening socket. It opens a UDP socket on the
// given port and polls for incoming packets.
type udpListener struct {
	errors  chan os.Error // Any network errors are available in here.
	in, out chan *datagram
	lock    *sync.Mutex
	conn    *net.UDPConn
}

func newUdpListener() *udpListener {
	l := new(udpListener)
	l.lock = new(sync.Mutex)
	return l
}

func (this *udpListener) IsOpen() bool { return this.conn != nil }

// Initialize and Run the listener.
func (this *udpListener) Run(addr *net.UDPAddr) (err os.Error) {
	if this.conn != nil {
		return
	}

	this.lock.Lock()
	if this.conn, err = net.ListenUDP("udp", addr); err != nil {
		return
	}

	this.errors = make(chan os.Error)
	this.in = make(chan *datagram, 8)
	this.out = make(chan *datagram, 8)

	this.lock.Unlock()

	go this.pollIn()
	go this.pollOut()
	return
}

// Close the listener
func (this *udpListener) Close() {
	this.lock.Lock()

	if this.conn != nil {
		this.conn.Close()
		this.conn = nil
	}

	time.Sleep(1e9) // give code some time to break out of loop

	if !closed(this.in) {
		close(this.in)
	}

	if !closed(this.out) {
		close(this.out)
	}

	if !closed(this.errors) {
		close(this.errors)
	}

	this.lock.Unlock()
}

// This polls the outgoing packet channel for data to be sent.
func (this *udpListener) pollOut() {
	var dg *datagram
	var err os.Error

	for this.conn != nil && !closed(this.out) {
		select {
		case dg = <-this.out:
			if _, err = this.conn.WriteToUDP(dg.Packet, dg.Addr); err != nil {
				this.errors <- err
			}
		}
	}
}

// This polls the listening socket for incoming packets.
func (this *udpListener) pollIn() {
	var err os.Error
	var size int
	var addr *net.UDPAddr
	var stamp int64

	datasize := PacketSize - 6 // = PacketSize-UdpHeader+len(ipv6(addr))
	data := make([]uint8, datasize, datasize)

	for this.conn != nil && !closed(this.in) && !closed(this.errors) {
		size, addr, err = this.conn.ReadFromUDP(data[16:]) // leave room for 16-byte address
		stamp = time.Nanoseconds()

		switch {
		case err != nil:
			this.errors <- err
		case size < 6: // Need 5 byte msg header + at least 1 byte data (msg id)
			this.errors <- ErrInvalidPacket
		default:
			copy(data, addr.IP.To16())
			this.in <- newDatagram(addr, data[0:size+16], stamp)
		}
	}
}
