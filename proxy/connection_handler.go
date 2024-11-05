package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

var (
	ErrNoAcceptableMethodsErr = errors.New("no acceptable methods")
	ErrIncorrectSocksVer      = errors.New("incorrect SOCKS5 version")
	ErrCMDIncorrect           = errors.New("command not supported / protocol error")
	ErrATYPIncorrect          = errors.New("address type not supported")
	ErrNetworkUnreachable     = errors.New("network unreachable")
	ErrHostUnreachable        = errors.New("host unreachable")
)

type ConnectionHandler struct {
	conn *net.TCPConn
}

func NewConnectionHandler(conn *net.TCPConn) *ConnectionHandler {
	return &ConnectionHandler{
		conn: conn,
	}
}

func (ch *ConnectionHandler) launch() error {
	conn := ch.conn
	defer func(conn *net.TCPConn) {
		_ = conn.Close()
		LOG.Debugln(conn.RemoteAddr().String(), "closed")
	}(conn)
	method, err := ch.handleHandshake()
	if err != nil {
		LOG.Errorln(conn.RemoteAddr(), "error occurred during handshake", err)
		return err
	}

	addr, atyp, err := ch.handleRequest()
	if err != nil {
		LOG.Errorln(
			conn.RemoteAddr(), "error occurred during handle initial request", err,
		)
		_ = ch.ReplyOnErrHandleRequest(err, atyp)
		return err
	}

	LOG.Debugln(addr.String(), "atyp", atyp, "method", method)
	remoteAddr, err := net.DialTCP("tcp", nil, addr)
	defer func(remoteAddr *net.TCPConn) { _ = remoteAddr.Close() }(remoteAddr)
	if err != nil {
		LOG.Errorln(addr.String(), "error occurred during dial", err)
		_ = ch.ReplyOnErrHandleRequest(err, atyp)
		return err
	}

	err = ch.ReplySuccessOnHandleRequest(remoteAddr, atyp)
	if err != nil {
		LOG.Errorln(addr.String(), "error occurred during success reply", err)
		return err
	}
	LOG.Traceln(
		"remote.remote:", remoteAddr.RemoteAddr(),
		"remote.local:", remoteAddr.LocalAddr(),
		"local.remote:", ch.conn.RemoteAddr(),
		"local.local:", ch.conn.LocalAddr(),
	)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { _ = copyData(conn, remoteAddr, &wg) }()
	_ = copyData(remoteAddr, conn, &wg)
	wg.Wait()
	LOG.Traceln("finish launch", err)
	return err
}

// Errors:
// - ErrIncorrectSocksVer : if message has incorrect protocol version
// - ErrNoAcceptableMethodsErr : if no acceptable methods in message
func (ch *ConnectionHandler) handleHandshake() (method byte, err error) {
	buf := make([]byte, 2)
	if _, err = io.ReadFull(ch.conn, buf); err != nil {
		return
	}

	if buf[0] != RequiredSocksVersion {
		LOG.Errorln()
		return AuthNoAcceptableMethods, ErrIncorrectSocksVer
	}

	methodsQ := int(buf[1])
	methods := make([]byte, methodsQ)
	if _, err = io.ReadFull(ch.conn, methods); err != nil {
		return
	}
	authMethod := AuthNoAcceptableMethods
	for _, method = range methods {
		if method == AuthMethodNoAuthenticationRequired {
			authMethod = method
		}
	}

	if authMethod == AuthNoAcceptableMethods {
		_, _ = ch.conn.Write([]byte{RequiredSocksVersion, AuthNoAcceptableMethods})
		return AuthNoAcceptableMethods, ErrNoAcceptableMethodsErr
	} else {
		_, err = ch.conn.Write([]byte{RequiredSocksVersion, authMethod})
	}
	if err != nil {
		return 0, err
	}

	return authMethod, nil
}

// errors:
// - ErrIncorrectSocksVer : if message has incorrect protocol version
// - ErrCMDIncorrect : if specified not supported command
// - ErrATYPIncorrect : if specified not supported address type
// - ErrNetworkUnreachable : if network unreachable
func (ch *ConnectionHandler) handleRequest() (
	remoteAddr *net.TCPAddr, atyp byte, err error) {
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	buf := make([]byte, 4)
	if _, err = io.ReadFull(ch.conn, buf); err != nil {
		return nil, 0, err
	}

	if buf[0] != RequiredSocksVersion {
		return nil, 0, ErrIncorrectSocksVer
	}

	cmd := buf[1]
	if cmd != CMDConnect {
		return nil, 0, ErrCMDIncorrect
	}

	var dstAddr string
	atyp = buf[3]
	if atyp == ATYPIPV4Address {
		addr := make([]byte, 4)
		port := make([]byte, 2)
		if _, err = io.ReadFull(ch.conn, addr); err != nil {
			return nil, atyp, err
		}
		if _, err = io.ReadFull(ch.conn, port); err != nil {
			return nil, atyp, err
		}
		dstAddr = fmt.Sprintf(
			"%s:%d", net.IP(addr).String(), binary.BigEndian.Uint16(port),
		)
	} else if atyp == ATYPDomainName {
		length := make([]byte, 1)
		if _, err = io.ReadFull(ch.conn, length); err != nil {
			return nil, atyp, err
		}
		domain := make([]byte, length[0])
		if _, err = io.ReadFull(ch.conn, domain); err != nil {
			return nil, atyp, err
		}
		port := make([]byte, 2)
		if _, err = io.ReadFull(ch.conn, port); err != nil {
			return nil, atyp, err
		}
		dstAddr = fmt.Sprintf("%s:%d", domain, binary.BigEndian.Uint16(port))
	} else {
		return nil, atyp, ErrATYPIncorrect
	}

	remoteAddr, err = net.ResolveTCPAddr("tcp", dstAddr)
	if err != nil {
		return nil, atyp, ErrNetworkUnreachable
	}
	LOG.Traceln("ResolveTCPAddr", remoteAddr.String())
	return remoteAddr, atyp, nil
}

func (ch *ConnectionHandler) ReplyOnErrHandleRequest(suppliedError error,
	adrType byte) (err error) {
	conn := ch.conn
	reply := []byte{RequiredSocksVersion, 0, 0x00, ATYPIPV4Address, 0, 0, 0, 0, 0, 0}
	if errors.Is(suppliedError, ErrNetworkUnreachable) {
		reply[1] = REPLYNetworkUnreachable
	} else if errors.Is(suppliedError, ErrATYPIncorrect) {
		reply[1] = REPLYAddressTypeNotSupported
	} else if errors.Is(suppliedError, ErrCMDIncorrect) {
		reply[1] = REPLYCommandNotSupported
	} else if errors.Is(suppliedError, ErrHostUnreachable) {
		reply[1] = REPLYHostUnreachable
	} else {
		reply[1] = REPLYConnectionRefused
	}
	_, suppliedError = conn.Write(reply)
	return err
}

func (ch *ConnectionHandler) ReplySuccessOnHandleRequest(
	dstConn *net.TCPConn, adrType byte) (err error) {
	localAddr := dstConn.LocalAddr().(*net.TCPAddr)
	response := []byte{
		RequiredSocksVersion, REPLYSucceeded, 0x00, ATYPIPV4Address,
		localAddr.IP[0], localAddr.IP[1], localAddr.IP[2], localAddr.IP[3], // BND.ADDR
		byte(localAddr.Port >> 8), byte(localAddr.Port & 0xFF), // BND.PORT
	}
	LOG.Traceln(response)
	_, err = ch.conn.Write(response)
	return err
}

func copyData(src *net.TCPConn, dest *net.TCPConn, wg *sync.WaitGroup) (err error) {
	defer wg.Done()
	defer func(dest *net.TCPConn) { _ = dest.CloseWrite() }(dest)
	_, err = io.Copy(src, dest)
	if err != nil {
		LOG.Errorln(": Reading error: ", err)
	}
	return err
}
