package wg

import (
	"io"
	"strconv"
	"sync"
)

var (
	headerStatic = []byte("[Interface]\nPrivateKey=")
	addrStatic   = []byte("\nAddress=10.5.0.2/16\nDNS=")
	peerStatic   = []byte("\n\n[Peer]\nPublicKey=")
	allowStatic  = []byte("\nAllowedIPs=0.0.0.0/0,::/0\nEndpoint=")
	portStatic   = []byte(":51820\nPersistentKeepalive=")

	pool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 0, 1024)
			return &b
		},
	}
)

func BuildPeerPrefix(pubKey string, endpoint []byte) []byte {
	size := len(peerStatic) + len(pubKey) + len(allowStatic) + len(endpoint) + len(portStatic)
	buf := make([]byte, 0, size)
	buf = append(buf, peerStatic...)
	buf = append(buf, pubKey...)
	buf = append(buf, allowStatic...)
	buf = append(buf, endpoint...)
	buf = append(buf, portStatic...)
	return buf
}

func Build(privateKey, dns string, peerPrefix []byte, keepAlive int) []byte {
	size := len(headerStatic) + len(privateKey) + len(addrStatic) + len(dns) + len(peerPrefix) + 5
	buf := make([]byte, 0, size)
	buf = append(buf, headerStatic...)
	buf = append(buf, privateKey...)
	buf = append(buf, addrStatic...)
	buf = append(buf, dns...)
	buf = append(buf, peerPrefix...)
	buf = strconv.AppendInt(buf, int64(keepAlive), 10)
	return buf
}

func WriteConfig(w io.Writer, privateKey, dns string, peerPrefix []byte, keepAlive int) error {
	bufPtr := pool.Get().(*[]byte)
	buf := *bufPtr
	buf = buf[:0]
	buf = append(buf, headerStatic...)
	buf = append(buf, privateKey...)
	buf = append(buf, addrStatic...)
	buf = append(buf, dns...)
	buf = append(buf, peerPrefix...)
	buf = strconv.AppendInt(buf, int64(keepAlive), 10)
	_, err := w.Write(buf)
	*bufPtr = buf
	pool.Put(bufPtr)
	return err
}
