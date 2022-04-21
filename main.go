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
	Channels []string `arg:"-c,required"`
	User     string   `arg:"-u,required"`
	Token    string   `arg:"-t,required"`
	Addr     string   `default:"192.168.1.23:9100"`
}

func main() {
	arg.MustParse(&args)

	conn, err := net.Dial("tcp", args.Addr)
	if err != nil {
		fmt.Println("unable to dial:", err)
		return
	}
	defer conn.Close()

	printer := hoin.NewPrinter(conn)
	defer func() {
		printer.Println("Closing client")
		printer.CutFeed(100)
	}()

	printer.Initialize()

	printer.Printf("Joining %s\n", args.Channels)

	log.SetOutput(io.MultiWriter(os.Stdout, printer))

	log.SetPrefix("\n  ")
	log.SetFlags(log.Lmsgprefix | log.LstdFlags)

	fmt.Println("Press ctrl+c to shutdown")
	ch := make(chan os.Signal, 5)
	signal.Notify(ch, os.Interrupt)

	go connectClient(printer, ch)

	<-ch
}

func connectClient(printer hoin.Printer, ch chan os.Signal) {
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

		status, err := printer.TransmitPaperSensorStatus()
		if err != nil {
			log.Fatalf("Could not get paper sensor status: %v\n", err)
		}
		if status.NearEnd || status.RollEnd {
			log.Println("Out of paper")
			ch <- os.Kill
		}
	})

	client.Join(args.Channels...)

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
