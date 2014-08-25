package manners

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"reflect"
	"sync"
	"syscall"
)

// NewListener wraps an existing listener for use with
// GracefulServer.
//
// Note that you generally don't need to use this directly as
// GracefulServer will automatically wrap any non-graceful listeners
// supplied to it.
func NewListener(l net.Listener) *GracefulListener {
	return &GracefulListener{
		listener: l,
		mutex:    &sync.RWMutex{},
		open:     true,
	}
}

// A gracefulCon wraps a normal net.Conn and tracks the
// last known http state.
type gracefulConn struct {
	net.Conn
	lastHTTPState http.ConnState
}

// A GracefulListener differs from a standard net.Listener in one way: if
// Accept() is called after it is gracefully closed, it returns a
// listenerAlreadyClosed error. The GracefulServer will ignore this
// error.
type GracefulListener struct {
	listener net.Listener
	open     bool
	mutex    *sync.RWMutex
}

func (l *GracefulListener) isClosed() bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return !l.open
}

func (l *GracefulListener) Addr() net.Addr {
	return l.listener.Addr()
}

// Accept implements the Accept method in the Listener interface.
func (l *GracefulListener) Accept() (net.Conn, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		if l.isClosed() {
			err = listenerAlreadyClosed{err}
		}
		return nil, err
	}

	gconn := &gracefulConn{conn, 0}
	return gconn, nil
}

// Close tells the wrapped listener to stop listening.  It is idempotent.
func (l *GracefulListener) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if !l.open {
		return nil
	}
	l.open = false
	return l.listener.Close()
}

func (l *GracefulListener) GetFD() (uintptr, string) {
	v := reflect.ValueOf(l.listener).Elem().FieldByName("fd").Elem()
	fd := uintptr(v.FieldByName("sysfd").Int())
	addr := l.listener.Addr()
	name := fmt.Sprintf("%s:%s->", addr.Network(), addr.String())
	return fd, name
}

func (l *GracefulListener) Clone() (*GracefulListener, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.open {
		return nil, fmt.Errorf("listener is already closed")
	}

	fd, fdName := l.GetFD()

	fl, err := net.FileListener(os.NewFile(fd, fdName))
	if nil != err {
		return nil, err
	}

	switch fl.(type) {
	case *net.TCPListener, *net.UnixListener:
	default:
		return nil, fmt.Errorf("file descriptor is %T not *net.TCPListener or *net.UnixListener", l)
	}
	if err := syscall.Close(int(fd)); nil != err {
		return nil, err
	}
	return NewListener(fl), nil
}

type listenerAlreadyClosed struct {
	error
}
