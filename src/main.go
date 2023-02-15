package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexflint/go-arg"
)

var args struct {
	Token              string `arg:"-t,--token" help:"telegram bot token"`
	TokenEnvKey        string `arg:"--token-env-key" help:"telegram bot token env key"`
	FirebaseConf       string `arg:"-c,--conf" help:"firebase service account base64 conf"`
	FirebaseConfEnvKey string `arg:"--conf-env-key" help:"firebase service account base64 conf env key"`
}

func launch() {
	InitDatabase()
	InitSession()
	InitMonitor()
	InitContext()
}

func main() {
	arg.MustParse(&args)

	if len(args.Token) != 0 {
		token = args.Token
	} else if len(args.TokenEnvKey) != 0 {
		token = os.Getenv(args.TokenEnvKey)
	}
	if len(token) == 0 {
		log.Fatal("token not found")
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	go launch()

	<-sigs
}
