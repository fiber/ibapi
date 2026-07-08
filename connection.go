/* connection handle the tcp socket to the TWS or IB Gateway*/

package ibapi

import (
	"net"
	"strconv"
	"sync/atomic"

	"go.uber.org/zap"
)

// IbConnection wrap the tcp connection with TWS or Gateway.
//
// Byte/message counters are updated on every socket op from Write
// (goRequest goroutine) and Read (goReceive goroutine) — they need
// to be atomic. The state field is read from goRequest's IsConnected
// check while HandShake / Disconnect write it from the main goroutine.
type IbConnection struct {
	*net.TCPConn
	host         string
	port         int
	clientID     int64
	state        atomic.Int32
	numBytesSent atomic.Int64
	numMsgSent   atomic.Int64
	numBytesRecv atomic.Int64
	numMsgRecv   atomic.Int64
}

func (ibconn *IbConnection) Write(bs []byte) (int, error) {
	n, err := ibconn.TCPConn.Write(bs)

	ibconn.numBytesSent.Add(int64(n))
	ibconn.numMsgSent.Add(1)

	log.Debug("conn write", zap.Int("nBytes", n))

	return n, err
}

func (ibconn *IbConnection) Read(bs []byte) (int, error) {
	n, err := ibconn.TCPConn.Read(bs)

	ibconn.numBytesRecv.Add(int64(n))
	ibconn.numMsgRecv.Add(1)

	log.Debug("conn read", zap.Int("nBytes", n))

	return n, err
}

func (ibconn *IbConnection) setState(state int32) {
	ibconn.state.Store(state)
}

func (ibconn *IbConnection) reset() {
	ibconn.numBytesSent.Store(0)
	ibconn.numBytesRecv.Store(0)
	ibconn.numMsgSent.Store(0)
	ibconn.numMsgRecv.Store(0)
}

func (ibconn *IbConnection) disconnect() error {
	log.Debug("conn disconnect",
		zap.Int64("nMsgSent", ibconn.numMsgSent.Load()),
		zap.Int64("nBytesSent", ibconn.numBytesSent.Load()),
		zap.Int64("nMsgRecv", ibconn.numMsgRecv.Load()),
		zap.Int64("nBytesRecv", ibconn.numBytesRecv.Load()),
	)
	// A freshly-constructed IbConnection has a nil embedded TCPConn.
	// Calling Close on it would nil-deref, so let Disconnect-before-
	// Connect resolve as a no-op instead of crashing.
	if ibconn.TCPConn == nil {
		return nil
	}
	return ibconn.Close()
}

func (ibconn *IbConnection) connect(host string, port int) error {
	var err error
	var addr *net.TCPAddr
	ibconn.host = host
	ibconn.port = port
	ibconn.reset()

	server := ibconn.host + ":" + strconv.Itoa(port)
	if addr, err = net.ResolveTCPAddr("tcp4", server); err != nil {
		log.Error("failed to resove tcp address", zap.Error(err), zap.String("host", server))
		return err
	}

	if ibconn.TCPConn, err = net.DialTCP("tcp4", nil, addr); err != nil {
		log.Error("failed to dial tcp", zap.Error(err), zap.Any("address", addr))
		return err
	}

	log.Debug("tcp socket connected", zap.Any("address", ibconn.TCPConn.RemoteAddr()))

	return err
}
