package network

// This interface represents a generic encryption algorythm. Any implementation
// allows us to (en/de)crypt packet data. The peerid string should not be
// considered a (en/de)cryption key. It is merely meant as a means to identify
// the source/target of the data and can be used as an index into another
// datasource which does hold key data specific to this client.
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
