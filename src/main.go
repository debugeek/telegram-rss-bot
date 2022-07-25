package main

import (
	"encoding/base64"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexflint/go-arg"
	"google.golang.org/api/option"
)

var args struct {
	Token              string `arg:"-t,--token" help:"telegram bot token"`
	TokenEnvKey        string `arg:"--token-env-key" help:"telegram bot token env key"`
	FirebaseConf       string `arg:"-c,--conf" help:"firebase service account base64 conf"`
	FirebaseConfEnvKey string `arg:"--conf-env-key" help:"firebase service account base64 conf env key"`
}

func launch() {
	InitFirebase()
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

	var conf []byte
	if len(args.FirebaseConf) != 0 {
		conf, _ = base64.StdEncoding.DecodeString(args.FirebaseConf)
	} else if len(args.FirebaseConfEnvKey) != 0 {
		conf, _ = base64.StdEncoding.DecodeString(os.Getenv(args.FirebaseConfEnvKey))
	} else {
		panic("firebase credential not found")
	}
	opt = option.WithCredentialsJSON(conf)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	go launch()

	<-sigs
}
