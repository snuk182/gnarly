package network

import "os"

var (
	ErrPacketSize      = os.NewError("Data size exceeds maximum Packet size")
	ErrInvalidPacket   = os.NewError("Unrecognized packet format")
	ErrInvalidClientID = os.NewError("Invalid Clientid specified")
)
