package xray

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var ErrBufferFull = errors.New("send buffer is full")
var Header = []byte(`{"format": "json", "version": 1}` + "\n")

type UDPCollector struct {
	Addr   string
	mu     sync.Mutex
	spans  chan []byte
	stopch chan struct{}
	stop   int32
	wg     sync.WaitGroup
}

func (u *UDPCollector) Send(span *Span) error {
	b, err := span.Encode()
	if err != nil {
		return err
	}

	select {
	case u.spans <- b:
	default:
		return ErrBufferFull
	}

	return nil
}

func NewUDPCollector(addr string) *UDPCollector {
	u := &UDPCollector{
		Addr:   addr,
		spans:  make(chan []byte, 1024),
		stopch: make(chan struct{}),
	}
	go u.connect()
	return u
}

func (u *UDPCollector) Stop() {
	if atomic.CompareAndSwapInt32(&u.stop, 0, 1) {
		close(u.stopch)
		u.wg.Wait()
	}
}

func (u *UDPCollector) Close() error {
	u.Stop()
	return nil
}

func (u *UDPCollector) connect() {
	u.wg.Wait()
	for atomic.LoadInt32(&u.stop) == 0 {
		if c, err := net.Dial("udp", u.Addr); err == nil {
			u.wg.Add(1)
			go u.SendLoop(c)
			return
		}
		time.Sleep(time.Second)
	}

}

func (u *UDPCollector) SendLoop(c net.Conn) {
	defer func() {
		if c != nil {
			c.Close()
		}
	}()
	defer u.wg.Done()

	for atomic.LoadInt32(&u.stop) == 0 {
		select {
		case b := <-u.spans:
			if _, err := c.Write(append(Header, b...)); err != nil {
				go u.connect()
				return
			}
		case <-u.stopch:
			return
		}
	}
}
