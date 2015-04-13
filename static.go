package manners

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	DefaultGracePeriod = 15 * time.Second
)

var (
	servers []*GracefulServer
	m       sync.Mutex
	signals sync.Once
)

// ListenAndServe provides a graceful version of the function provided by the net/http package.
// Call Close() to stop all started servers.
func ListenAndServe(addr string, handler http.Handler) error {
	registerShutdownSignals()
	server := NewWithServer(&http.Server{Addr: addr, Handler: handler})
	m.Lock()
	servers = append(servers, server)
	m.Unlock()
	return server.ListenAndServe()
}

// ListenAndServeTLS provides a graceful version of the function provided by the net/http package.
// Call Close() to stop all started servers.
func ListenAndServeTLS(addr string, certFile string, keyFile string, handler http.Handler) error {
	registerShutdownSignals()
	server := NewWithServer(&http.Server{Addr: addr, Handler: handler})
	m.Lock()
	servers = append(servers, server)
	m.Unlock()
	return server.ListenAndServeTLS(certFile, keyFile)
}

// Serve provides a graceful version of the function provided by the net/http package.
// Call Close() to stop all started servers.
func Serve(l net.Listener, handler http.Handler) error {
	registerShutdownSignals()
	server := NewWithServer(&http.Server{Handler: handler})
	m.Lock()
	servers = append(servers, server)
	m.Unlock()
	return server.Serve(l)
}

// Close triggers a shutdown of all running Graceful servers.
// Call Close() to stop all started servers.
func Close() {
	m.Lock()
	for _, s := range servers {
		s.Close()
	}
	servers = nil
	m.Unlock()
}

func registerShutdownSignals() {
	signals.Do(func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT)
		go func() {
			<-sigc
			log.Printf("Beginning grace period (%s)\n", DefaultGracePeriod)
			select {
			// If a second signal is received, abandon ship and exit immediatly
			case <-sigc:
				log.Println("Canceled grace period")
			case <-time.After(DefaultGracePeriod):
			}
			Close()
			log.Println("Listeners closed")
		}()
	})
}
