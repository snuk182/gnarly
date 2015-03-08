package network

import "errors"

var (
	ErrInvalidClientID       = errors.New("Invalid Clientid specified")
	ErrInvalidPacket         = errors.New("Invalid packet format")
	ErrInvalidMessageHandler = errors.New("Invalid message handler")
	ErrInvalidErrorHandler   = errors.New("Invalid error handler")
	ErrPacketSequenceTooLong = errors.New("Packet Sequence too long (>65535)")
	ErrNoData                = errors.New("No data in packet.")
)
