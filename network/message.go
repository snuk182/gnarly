package network

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

// This is a message sent to the host application. It contains the data and
// peer ID of one or more packets.
type Message struct {
	Type   uint8
	PeerId string
	Data   []uint8
}

func NewMessage(mt uint8, peerid string, data []uint8) *Message {
	m := new(Message)
	m.Type = mt
	m.PeerId = peerid
	m.Data = data
	return m
}
