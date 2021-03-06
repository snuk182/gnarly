package network

// Message types
const (
	MsgData             uint8 = iota // Generic data packet. No specific message implied.
	MsgPing                          // Represents a Ping message.
	MsgPong                          // Response to Ping message
	MsgPeerConnected                 // A new peer has been detected
	MsgPeerDisconnected              // A known peer has timed out.
	MsgLatency                       // Reports a client's latency at customizable intervals

	// Dummy value. Used to indicate where a host application should start
	// defining it's own message types. MsgMax, MsgMax+1, MsgMax+2 etc.
	// There is room for 200 custom message types: 55-255
	MsgMax uint8 = 55
)
