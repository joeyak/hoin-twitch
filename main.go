package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/alexflint/go-arg"
	"github.com/gempir/go-twitch-irc/v3"
	"github.com/joeyak/hoin-printer"
)

var args struct {
	SkipPrinter bool     `arg:"--skip"`
	Channels    []string `arg:"-c,required"`
	User        string   `arg:"-u,required"`
	Token       string   `arg:"-t,required"`
	Addr        string   `default:"192.168.1.23:9100"`
}

func main() {
	arg.MustParse(&args)

	if !args.SkipPrinter {
		conn, err := net.Dial("tcp", args.Addr)
		if err != nil {
			fmt.Println("unable to dial:", err)
			return
		}
		defer conn.Close()

		printer := hoin.NewPrinter(conn)
		defer func() {
			printer.LF()
			printer.Println("Closing client")
			printer.CutFeed(100)
		}()

		printer.Printf("Joining %s\n", args.Channels)

		log.SetOutput(io.MultiWriter(os.Stdout, printer))
	}

	log.SetPrefix("\n  ")
	log.SetFlags(log.Lmsgprefix | log.LstdFlags)

	go connectClient()

	fmt.Println("Press ctrl+c to shutdown")
	ch := make(chan os.Signal, 5)
	signal.Notify(ch, os.Interrupt)
	<-ch
}

func connectClient() {
	client := twitch.NewClient(args.User, args.Token)

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if message.Type != twitch.PRIVMSG {
			log.Println(message.Type, message.Message)
			return
		}

		if len(args.Channels) > 1 {
			log.Printf("[%s] %s: %s\n", message.Channel, message.User.DisplayName, message.Message)
		} else {
			log.Printf("%s: %s\n", message.User.DisplayName, message.Message)
		}
	})

	client.Join(args.Channels...)

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
