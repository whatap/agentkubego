package pool
//customized  https://github.com/fatih/pool
import (
	"errors"
	"fmt"
	"sync"
)


var (
	ErrClosed = errors.New("pool is closed")
)

type Connection interface{
	Close() error
}

type Pool interface {
	Get() (Connection, error)
	Put(c Connection) (error)

	Close()

	Len() int
}

type channelPool struct {
	mu    sync.RWMutex
	conns chan Connection
	factory Factory
}

type Factory func() (Connection, error)

func NewChannelPool(initialCap, maxCap int, factory Factory) (Pool, error) {

	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	c := &channelPool{
		conns:   make(chan Connection, maxCap),
		factory: factory,
	}

	var conns []Connection
	for i := 0; i < initialCap; i++ {
		conn, err := factory()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}

		conns = append(conns, conn)
	}

	return c, nil
}



func (c *channelPool) getConnsAndFactory() (chan Connection, Factory) {

	c.mu.RLock()

	conns := c.conns

	factory := c.factory

	c.mu.RUnlock()

	return conns, factory

}

func (c *channelPool) Get() (Connection, error) {
	conns, _ := c.getConnsAndFactory()

	if conns == nil {
		return nil, ErrClosed
	}
	conn := <-conns
	return conn, nil

	// select {
	// case conn := <-conns:
	// 	if conn == nil {
	// 		return nil, ErrClosed
	// 	}

	// 	return conn, nil

	// default:
	// 	conn, err := factory()

	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	return conn, nil
	// }
}

func (c *channelPool) Put(conn Connection) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conns == nil {
		return conn.Close()
	}

	select {
	case c.conns <- conn:
		return nil

	default:
		return conn.Close()
	}
}

func (c *channelPool) Close() {
	c.mu.Lock()
	conns := c.conns
	c.conns = nil
	c.factory = nil
	c.mu.Unlock()

	if conns == nil {
		return

	}
	close(conns)

	for conn := range conns {
		conn.Close()
	}
}

func (c *channelPool) Len() int {
	conns, _ := c.getConnsAndFactory()
	return len(conns)
}


