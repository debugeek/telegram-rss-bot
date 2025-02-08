package main

import (
	"os"
	"os/signal"
	"syscall"
)

var app *App

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	app = &App{}
	go app.launch()

	<-sigs
}
