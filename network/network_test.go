package network

import "testing"
import "net"
import "crypto/md5"
import "encoding/base64"
import "bytes"

func TestPeerIdIPv4(t *testing.T) {
	a := getPeerId(t, "80.254.11.3", "192.168.2.101")
	b := getPeerId(t, "80.254.11.3", "192.168.2.102")

	if a == b {
		t.Errorf("Peer IDs should not be identical. %#v vs %#v", a, b)
	}
}

func TestPeerIdIPv6(t *testing.T) {
	a := getPeerId(t, "80.254.11.3", "fe80::222:15ff:fe65:b2f9")
	b := getPeerId(t, "80.254.11.3", "fe80::222:15ff:fe65:b2fa")

	if a == b {
		t.Errorf("Peer IDs should not be identical. %#v vs %#v", a, b)
	}
}

func getPeerId(t *testing.T, public, private string) string {
	var ip net.IP
	if ip = net.ParseIP(private).To16(); ip == nil {
		t.Errorf("Invalid local IP address: %v", private)
		t.FailNow()
		return ""
	}

	var d []uint8
	buf := bytes.NewBuffer(d)
	buf.Write(net.ParseIP(public).To16())
	buf.Write([]byte{ip[14], ip[15]})

	h := md5.Sum(buf.Bytes())
	hash := h[0:16]

	buf.Truncate(0)
	enc := base64.NewEncoder(base64.StdEncoding, buf)

	if _, err := enc.Write(hash); err != nil {
		enc.Close()
		return ""
	}

	enc.Close()
	return buf.String()
}
