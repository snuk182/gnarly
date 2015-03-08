package network

import (
	"crypto/md5"
)

func md5hash(data []byte) []byte {
	h := md5.New()
	h.Write(data)
	return h.Sum(nil)
}
