package main

import (
	"context"
	"flag"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/Augiro/a2s-cache/cache"
	"github.com/Augiro/a2s-cache/poller"
	"github.com/Augiro/a2s-cache/server"
)

func setupLogger(debug bool) *zap.SugaredLogger {
	logLevel := zap.NewAtomicLevelAt(zap.InfoLevel)
	if debug {
		logLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	config := zap.NewDevelopmentConfig()
	config.Level = logLevel
	config.Development = false

	return zap.Must(config.Build()).Sugar()
}

func main() {
	debug := flag.Bool("debug", false, "enable debug logs")
	gIP := flag.String("gameIP", "1.2.3.4", "IP of the game server")
	gPort := flag.Int("gamePort", 27015, "port for the game server")
	host := flag.String("ip", "127.0.0.1", "IP that UDP server should listen on")
	port := flag.Int("port", 9000, "port UDP server should listen on")
	flag.Parse()

	log := setupLogger(*debug)
	defer log.Sync()

	group, ctx := errgroup.WithContext(context.Background())

	// Set up the cache.
	c := cache.New()

	// Setup & start the poller.
	p := poller.New(log, *gIP, *gPort, c)
	group.Go(func() error { p.Start(ctx); return nil })

	// Setup & start the UDP server.
	s := server.New(log, *host, *port, c)
	group.Go(func() error { return s.Start(ctx) })

	// Panic if any of the goroutines in the group fail.
	err := group.Wait()
	if err != nil {
		panic(err)
	}
}
