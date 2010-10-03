package network

// This type represents a function handler for dealing with incoming messages.
type MessageHandler func(client *Peer, msgtype uint8, data []uint8)

// Message types
const (
	MsgData             uint8 = iota // Generic data packet. No specific message implied.
	MsgPing                          // Represents a Ping message.
	MsgPong                          // Response to Ping message
	MsgPeerConnected                 // A new peer has been detected
	MsgPeerDisconnected              // A known peer has timed out.

	// Dummy value. Used to indicate where a host application should start 
	// defining it's own message types. MsgMax, MsgMax+1, MsgMax+2 etc.
	// There is room for 100 custom message types.
	MsgMax uint8 = 0x9b // 255-100
)
