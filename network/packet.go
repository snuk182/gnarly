package network

import "crypto/md5"
import "fmt"

// Packet flags
const (
	PFCompressed uint8 = 1 << iota // Indicates the packet is compressed.
	PFEncrypted                    // Tells us that the packet content is encrypted.
	PFFragmented                   // This tells us the packet is 1 part of a larger dataset.
)

// Represents a individual UDP packet. Fields in a packet byte slice listed in
// order of appearance:
//
// > Part of the UDP header:
//   - Address, 16 bytes
//
// > Header section:
//   - ClientId, 2 bytes
//   - Flags, 1 byte
//   - Sequence, 2 bytes
//   - (optional) Subsequence, 2 bytes
//
// > Data section:
//   - Data, len(Packet) - 16 - len(header) bytes
type Packet []byte

func (this Packet) ClientId() []byte { return this[16:18] }
func (this Packet) Flags() uint8     { return this[18] }
func (this Packet) Sequence() uint16 { return uint16(this[19])<<8 | uint16(this[20]) }

func (this Packet) SubSequence() (uint8, uint8) {
	if this[18]&PFFragmented != 0 {
		return this[21], this[22]
	}
	return 0, 1
}

func (this Packet) Data() []byte {
	if this[18]&PFFragmented != 0 {
		return this[23:]
	}
	return this[21:]
}

func (this Packet) String() string {
	ss1, ss2 := this.SubSequence()
	id := this.ClientId()
	return fmt.Sprintf("[%03d/%03d] | 0x%02x | 0x%04x | %02x %02x | %#v",
		ss1, ss2, this.Flags(), this.Sequence(), id[0], id[1], string(this.Data()))
}

// This is a Md5 hash of the sender's IP and ClientID to and represents a 
// unique identifier for the sender. This should ensure accurate
// identification of a user, even from multiple clients behind a NAT router,
// or from a client behind a NAT router which arbitrarilly reassigns UDP 
// ports (rare edge case). It is not actually sent across the wire, but 
// rather it is generated from the receiver IP in the UDP header and the 2
// byte clientID in the message header. It is not to be confused with a
// session key. This Owner id is bound to the physical machine the user is
// currently operating on. It remains the same across all sessions from that
// particular machine, as long as it has the same public and private IP
// addresses.
// 
// Example: Bob and Joe share a LAN network through a NAT enabled router.
//   The public IP for both is: 123.123.123.123
//   Bob's private IP (within the LAN network) is: 192.168.1.1
//   Joe's private IP (within the LAN network) is: 192.168.1.2
// 
// Bob's owner id is calculated as follows:
//     V1 = to_ipv6("123.123.123.123")            // = 16 bytes
//     V2 = 192.168.1.1 & 0.0.255.255 = 1.1       // = 2 bytes
//    Bob = md5(V1 + V2)                          // = 16 bytes
//
// Joe's owner id is calculated as follows:
//     V1 = to_ipv6("123.123.123.123")            // = 16 bytes
//     V2 = 192.168.1.2 & 0.0.255.255 = 1.2       // = 2 bytes
//    Joe = md5(V1 + V2)                          // = 16 bytes
func (this Packet) Owner() string {
	hash := md5.New()
	hash.Write(this[0:18]) // 16 byte ipv6 address + 2 byte client id
	return string(hash.Sum())
}
