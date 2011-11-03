package network

import "os"
import "net"
import "strings"

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
// This value defaults to 576. While 1400 is the accepted standard for most
// modern applications, it does put dial-up users at a disadvantage. 576 bytes
// ensures maximum compatibility, but If you are not targeting these, we
// recommend changing this to 1400 byte.
var PacketSize int = 576

// When set, this will be used to (de)compress packet data if the appropriate
// flags are set and Compression != nil. While compresion is generally
// advisable, it can in some cases lead to slower performance without any real
// reduction in data size. Test this out with the data you intend to send in
// order to determine if you want compression or not. You can overwrite this
// with your own compression code by simply implementing the network.Compressor
// interface and assigning a new instance of that type to this variable. To
// disable compression, simply set this to nil.
var Compression Compressor = NewGnarlyCompression()

// When set, this will be used to (en/de)crypt packet data if the appropriate
// flags are set and Encryption != nil. You can overwrite this with your
// own encryption code by simply implementing the network.Encrypter interface
// and assigning a new instance of that type to this variable. To disable
// encryption, simply set this to nil.
var Encryption Encrypter = NewGnarlyEncryption()


// Creates a 2 byte ClientID from the local machine's IP.
func GetClientId() (id []byte, err os.Error) {
	var conn *net.UDPConn
	var addr *net.UDPAddr

	// Connect to a random machine somewhere in this subnet. It's irrelevant
	// where to, as long as it's not the loopback address.
	if addr, err = net.ResolveUDPAddr("udp", "192.168.1.1:0"); err != nil {
		return
	}

	if conn, err = net.DialUDP("udp", nil, addr); err != nil {
		return
	}

	defer conn.Close()

	// strip port number off.
	str := conn.LocalAddr().String()
	if idx := strings.LastIndex(str, ":"); idx != -1 {
		str = str[0:idx]
	}

	var ip net.IP
	if ip = net.ParseIP(str).To16(); ip == nil {
		return
	}

	// TODO(jimt): I am unsure how 2 full IPv6 addresses in the same subnet relate
	// to eachother. Specifically if the 2 last bytes in the 16-byte address are
	// really the relevant bits that set them apart from eachother.
	// For IPv4 this is simple: 192.168.2.101 vs 192.168.2.102 -> we need the 
	// '2.101' and '2.102' bits. Bytes are stored in Big Endian order.
	id = []byte{ip[14], ip[15]}
	return
}
