package proxy

import (
	"errors"
	"io"
	"net"
)

var (
	NoAcceptableMethodsErr = errors.New("no acceptable methods")
	IncorrectSocksVer      = errors.New("incorrect SOCKS5 version")
	IncorrectCommandType   = errors.New("handle only CONNECT 0x01")
	IncorrectATYP          = errors.New("handle only DomainName/IPV4Address")
)

type ConnectionHandler struct {
	conn *net.TCPConn
}

func NewConnectionHandler(conn *net.TCPConn) *ConnectionHandler {
	return &ConnectionHandler{
		conn: conn,
	}
}

func (ch ConnectionHandler) launch() error {
	defer func(conn *net.TCPConn) { _ = conn.Close() }(ch.conn)
	method, err := ch.handleHandshake()
	if err != nil {
		return err
	}

}

func (ch ConnectionHandler) handleHandshake() (method byte, err error) {
	buf := make([]byte, 2)
	if _, err = io.ReadFull(ch.conn, buf); err != nil {
		LOG.Errorln(
			"Connection:", ch.conn,
			": error reading the version and methods:", err,
		)
		return
	}

	if buf[0] != RequiredSocksVersion {
		LOG.Errorln()
		return
	}
	_, err = ch.conn.Write([]byte{0x05, 0x00})
	if err != nil {
		return
	}

	methodsQ := int(buf[1])
	methods := make([]byte, methodsQ)
	if _, err = io.ReadFull(ch.conn, methods); err != nil {
		LOG.Errorln(
			"Connection:", ch.conn,
			": error reading the methods:", err,
		)
		return
	}
	authMethod := AuthNoAcceptableMethods
	for _, method = range methods {
		if method == RequiredAuthMethod {
			authMethod = method
		}
	}
	if authMethod == AuthNoAcceptableMethods {
		_, _ = ch.conn.Write([]byte{RequiredSocksVersion, AuthNoAcceptableMethods})
		LOG.Infoln(
			"Connection:", ch.conn,
			": no acceptable methods provided",
		)
		return AuthNoAcceptableMethods, NoAcceptableMethodsErr
	}

	_, err = ch.conn.Write([]byte{RequiredSocksVersion, authMethod})
	if err != nil {
		LOG.Errorln("Connection:", ch.conn, err)
		return
	}
	return authMethod, nil
}

func (ch ConnectionHandler) handleConnectionReq() (tcpAddr *net.TCPAddr, err error) {
	buf := make([]byte, 4)
	if _, err = io.ReadFull(ch.conn, buf); err != nil {
		LOG.Errorln("Connection:", ch.conn, err)
		return nil, err
	}

	if buf[0] != RequiredSocksVersion {
		return nil, IncorrectSocksVer
	}
	cmd := buf[1]
	atyp := buf[3]
	if cmd != CMDConnect {
		return nil, IncorrectCommandType
	}
	if atyp != ATYPDomainName || atyp != ATYPIPV4Address {
		return nil, IncorrectATYP
	}
	addr := make([]byte, 4)
	if _, err := io.ReadFull(ch.conn, addr); err != nil {
		LOG.Errorln(ch.conn, ": error reading remote address: ", err)
		return
	}
	port := make([]byte, 2)
	if _, err := io.ReadFull(ch.conn, port); err != nil {
		LOG.Errorln(ch.conn, ": error reading the port: ", err)
		return
	}
}
