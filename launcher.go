package main

import (
	"socks5_proxy/proxy"
	"strconv"
)

type LauncherConfig struct {
	port int
}

type Launcher struct {
	config LauncherConfig
}

func NewLauncher(config LauncherConfig) *Launcher {
	return &Launcher{config}
}

func (l Launcher) launch() (err error) {
	port := strconv.Itoa(l.config.port)
	listener, err := proxy.NewListener(":" + port)
	if err != nil {
		proxy.LOG.Errorln("NewListener", err)
		return err
	}
	err = listener.Launch()
	if err != nil {
		proxy.LOG.Errorln("listener.Launch", err)
		return err
	}
	return nil
}
