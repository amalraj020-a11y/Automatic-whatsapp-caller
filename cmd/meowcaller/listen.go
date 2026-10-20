package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	meowcaller "github.com/purpshell/meowcaller"
	"github.com/rs/zerolog"
)

type listenConfig struct {
	storePath string
	tempDir   string
	message   string
}

func parseListenArgs(args []string, stderr io.Writer) (listenConfig, error) {
	var cfg listenConfig
	store, err := defaultStorePath()
	if err != nil {
		return cfg, err
	}
	cfg.storePath = store
	cfg.tempDir = os.TempDir()
	cfg.message = "I am Amala, Raj's AI assistant bot. You can message me on WhatsApp or call my boss if needed."
	f := flag.NewFlagSet("listen", flag.ContinueOnError)
	f.SetOutput(stderr)
	f.StringVar(&cfg.storePath, "store", cfg.storePath, "linked-device SQLite store")
	f.StringVar(&cfg.tempDir, "temp-dir", cfg.tempDir, "private temporary audio directory")
	f.StringVar(&cfg.message, "message", cfg.message, "identity message spoken to incoming callers")
	if err := f.Parse(args); err != nil {
		return cfg, errors.Join(errUsage, err)
	}
	if f.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: meowcaller listen [flags]")
		return cfg, errUsage
	}
	return cfg, nil
}

func runListen(ctx context.Context, cfg listenConfig, stderr io.Writer) error {
	log := zerolog.Ctx(ctx)
	session, err := openSession(ctx, cfg.storePath, *log)
	if err != nil {
		return err
	}
	defer session.Close()
	if err := session.ConnectPaired(ctx); err != nil {
		return err
	}

	client := meowcaller.NewClient(session.client, meowcaller.WithLogger(*log))
	client.OnIncomingCall(func(call *meowcaller.Call) {
		log.Info().Msg("incoming WhatsApp call received; answering with Amala identity message")
		if err := call.Answer(); err != nil {
			log.Error().Err(err).Msg("incoming call answer failed")
			return
		}
		src, err := synthesize(ctx, cfg.tempDir, cfg.message)
		if err != nil {
			log.Error().Err(err).Msg("identity message synthesis failed")
			_ = call.Hangup()
			return
		}
		player := call.Play(src)
		player.OnFinish(func() {
			log.Info().Msg("identity message finished; ending unknown-caller call")
			time.AfterFunc(500*time.Millisecond, func() { _ = call.Hangup() })
		})
	})

	log.Info().Msg("listening for incoming WhatsApp calls")
	<-ctx.Done()
	return nil
}
