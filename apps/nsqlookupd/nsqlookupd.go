package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bitly/nsq/nsqlookupd"
	"github.com/bitly/nsq/util"
	"github.com/kardianos/service"
	"github.com/mreiferson/go-options"
)

var (
	flagSet = flag.NewFlagSet("nsqlookupd", flag.ExitOnError)

	config      = flagSet.String("config", "", "path to config file")
	showVersion = flagSet.Bool("version", false, "print version string")
	verbose     = flagSet.Bool("verbose", false, "enable verbose logging")

	tcpAddress       = flagSet.String("tcp-address", "0.0.0.0:4160", "<addr>:<port> to listen on for TCP clients")
	httpAddress      = flagSet.String("http-address", "0.0.0.0:4161", "<addr>:<port> to listen on for HTTP clients")
	broadcastAddress = flagSet.String("broadcast-address", "", "address of this lookupd node, (default to the OS hostname)")

	inactiveProducerTimeout = flagSet.Duration("inactive-producer-timeout", 300*time.Second, "duration of time a producer will remain in the active list since its last ping")
	tombstoneLifetime       = flagSet.Duration("tombstone-lifetime", 45*time.Second, "duration of time a producer will remain tombstoned if registration remains")
)

type program struct {
	daemon *nsqlookupd.NSQLookupd
}

func (p *program) Start(s service.Service) error {
	flagSet.Parse(os.Args[1:])

	if *showVersion {
		fmt.Println(util.Version("nsqlookupd"))
		os.Exit(0)
		return nil
	}

	var cfg map[string]interface{}
	if *config != "" {
		_, err := toml.DecodeFile(*config, &cfg)
		if err != nil {
			log.Fatalf("ERROR: failed to load config file %s - %s", *config, err.Error())
		}
	}

	opts := nsqlookupd.NewNSQLookupdOptions()
	options.Resolve(opts, flagSet, cfg)
	p.daemon = nsqlookupd.NewNSQLookupd(opts)

	p.daemon.Main()

	return nil
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.

	if p.daemon != nil {
		signalChan := make(chan struct{})

		go func() {
			p.daemon.Exit()
			signalChan <- struct{}{}
		}()

		timeout, _ := time.ParseDuration("30s")
		time.AfterFunc(timeout, func() {
			log.Fatalf("ERROR: failed to stop nsqlookupd in %s", timeout)
		})

		<-signalChan
	}

	return nil
}

var logger service.Logger

func main() {
	svcConfig := &service.Config{
		Name:        "nsqlookupd",
		DisplayName: "nsqlookupd",
		Description: "nsqlookupd 0.3.2",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
