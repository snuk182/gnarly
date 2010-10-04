package network

import "os"

var (
	ErrInvalidClientID = os.NewError("Invalid Clientid specified")
	ErrInvalidPacket   = os.NewError("Invalid packet format")
)
