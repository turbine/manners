package manners

import (
	"net"
	"testing"
)

func TestListenerGetFD(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Failed to create a listener", err)
	}
	g := NewListener(l)
	fd, _ := g.GetFD()
	if fd == 0 {
		t.Fatal("Failed to get and FD", fd)
	}

	g.Close()
}
