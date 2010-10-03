package network

// This interface represents a generic encryption algorythm. Any implementation
// allows us to (en/de)crypt packet data.
type Encrypter interface {
	Encrypt(peerid string, data []uint8) []uint8
	Decrypt(peerid string, data []uint8) []uint8
}

// A simple implementation of the network.Encrypter interface.
type GnarlyEncryption struct{}

func NewGnarlyEncryption() *GnarlyEncryption {
	return new(GnarlyEncryption)
}

func (this GnarlyEncryption) Encrypt(peerid string, in []uint8) []uint8 {
	return in
}

func (this GnarlyEncryption) Decrypt(peerid string, in []uint8) []uint8 {
	return in
}
