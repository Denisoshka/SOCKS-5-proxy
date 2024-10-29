package proxy

import (
	"net"
)

type SOCKS5ProxyListener struct {
	addr *net.TCPAddr
}

func NewListener(listenAddr string) (l *SOCKS5ProxyListener, err error) {
	addr, err := net.ResolveTCPAddr("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	return &SOCKS5ProxyListener{addr: addr}, nil
}

func (l SOCKS5ProxyListener) Launch() (err error) {
	listener, err := net.ListenTCP("tcp", l.addr)
	if err != nil {
		LOG.Errorln(err)
		return err
	}
	LOG.Infoln("Listener work on ", listener.Addr().String())
	defer func(listener *net.TCPListener) { _ = listener.Close() }(listener)
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			LOG.Errorln(err)
			continue
		}
		LOG.Debugf("Accepted new connection from %s", conn.RemoteAddr())
		go func() { _ = NewConnectionHandler(conn).launch() }()
	}
}
