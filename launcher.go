package main

type LauncherConfig struct {
	port int
}

type Launcher struct {
	config LauncherConfig
}

func NewLauncher(config LauncherConfig) *Launcher {
	return &Launcher{config}
}

func (l Launcher) launch() {

}
