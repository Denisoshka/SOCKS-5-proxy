package main

import "socks5_proxy/proxy"

func main() {
	port := proxy.ParseCLI()
	config := LauncherConfig{port: port}
	_ = NewLauncher(config).launch()
}
