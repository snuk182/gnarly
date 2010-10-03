package network

// This is a message sent to the host application. It contains the data and
// peer ID of one or more packets.
type Message struct {
	PeerId string
	Data   []byte
}

func NewMessage(peerid string, data []byte) *Message {
	m := new(Message)
	m.PeerId = peerid
	m.Data = data
	return m
}
