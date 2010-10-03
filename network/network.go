package network

import "os"
import "net"

// This is the size of a standard UDP datagram header. it is part of every
// packet we send. This header is processed by the operating system's transport
// layer. We will never see it. Note that this header counts towards the maximum
// size for a single UDP datagram (see: network.PacketSize)
const UdpHeaderSize = 22

// Maximum size of individual packets in bytes.
// Some established defaults are listed here:
//
// * 1500 - The largest Ethernet packet size; it is also the default value. This 
//          is the typical setting for non-PPPoE, non-VPN connections. The
//          default value for NETGEAR routers, adapters and switches.
// * 1492 - The size PPPoE prefers.
// * 1472 - Maximum size to use for pinging. (Bigger packets are fragmented.)
// * 1468 - The size DHCP prefers.
// * 1460 - Usable by AOL if you don't have large email attachments, etc.
// * 1430 - The size VPN and PPTP prefer.
// * 1400 - Maximum size for AOL DSL.
// *  576 - Typical value to connect to dial-up ISPs. (Default)
//
// This value defaults to 1400. This generally seems to be accepted as the most
// practical default size. It's slightly smaller than a typical Ethernet MTU
// (Maximum Transmission Unit) and still allows a rather sizable chunk of data
// to be transfered without the need for fragmentation.
var PacketSize int = 1400

// Used in conjuction with network.PacketLoss. it allows us to arbitrarilly
// drop a percentage of incoming data to simulate lag or a crappy connection.
// This is strictly a debugging/testing feature for development purposes. Do not
// use this in production code.
var SimulatePacketloss bool = false

// Percentage (0-100) of packets that should be considered 'lost'. This is
// strictly a debugging/testing feature to simulate lag. Leave this at 0 for
// production code. This will pick random packages up to a given percentage of
// the total amount of received packets and simply discard them. To enable this
// feature, set network.SimulatePacketloss to true.
var Packetloss byte = 0

// This boolean indicates whether we should compress packet data or not. While
// this is generally advisable, it can in some cases lead to slower performance
// without any real reduction in data size. Test this out with the data you
// intend to send in order to determine if you want compression or not. Default
// is 'true'. If set to true, this calls network.Compress and network.Decompress
// functions to do the actual work. You can overwrite these handlers with your
// own compression handlers.
var Compressed bool = true

type CompressionHandler func([]byte) []byte
type DecompressionHandler func([]byte) []byte

// The compression handler. overwrite this with your own compression handler if
// you do not wish to use the default compression routines.
var Compress CompressionHandler = func(in []byte) []byte {
	return in
}

// The decompression handler. overwrite this with your own compression handler
// if you do not wish to use the default compression routines.
var Decompress DecompressionHandler = func(in []byte) []byte {
	return in
}

// When set, will call the network.Encrypt and network.Decrypt functions to do
// the actual transformations. Note that by default these not set.
// You should assign your own encryption handlers to these variables to perform
// the actual transformation according to the scheme of your choice.
var Encrypted bool = true

type EncryptionHandler func(owner, data []byte) ([]byte, os.Error)
type DecryptionHandler func(owner, data []byte) ([]byte, os.Error)

// This variable should be set to a valid encryption handler if network.Encrypted = true/
var Encrypt EncryptionHandler

// This variable should be set to a valid decryption handler if network.Encrypted = true
var Decrypt DecryptionHandler

// Creates a 2 byte ClientID from the given IP address.
func GetClientId(addr string) (id []byte, err os.Error) {
	if len(addr) < 3 {
		return nil, ErrInvalidClientID
	}

	if addr[0] == '[' {
		addr = addr[1:]
	}

	if addr[len(addr)-1] == ']' {
		addr = addr[0 : len(addr)-1]
	}

	var ip net.IP
	if ip = net.ParseIP(addr).To16(); ip == nil {
		return nil, ErrInvalidClientID
	}

	// TODO(jimt): I am unsure how 2 full IPv6 addresses in the same subnet relate
	// to eachother. Specifically if the 2 last bytes in the 16-byte address are
	// really the relevant bits that set them apart from eachother.
	// For IPv4 this is simple: 192.168.2.101 vs 192.168.2.102 -> we need the 
	// '2.101' and '2.102' bits. Bytes are stored in Big Endian order.
	id = []byte{ip[14], ip[15]}
	return
}
