package network

// This interface represents a generic compression algorythm. Any implementation
// allows us to (de)compress packet data.
type Compressor interface {
	Compress(data []uint8) []uint8
	Decompress(data []uint8) []uint8
}

// A simple implementation of the network.Compressor interface.
type GnarlyCompression struct{}

func NewGnarlyCompression() *GnarlyCompression {
	return new(GnarlyCompression)
}

func (this GnarlyCompression) Compress(in []uint8) []uint8 {
	return in
}

func (this GnarlyCompression) Decompress(in []uint8) []uint8 {
	return in
}
