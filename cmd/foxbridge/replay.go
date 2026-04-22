package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/VulpineOS/foxbridge/pkg/recording"
)

func runReplayCommand(args []string) int {
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	port := fs.Int("port", 9222, "CDP WebSocket port")
	socket := fs.String("socket", "", "Unix-domain socket path for the CDP HTTP/WebSocket server")
	recordingPath := fs.String("recording", "", "Path to a foxbridge JSONL recording")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *recordingPath == "" {
		fmt.Fprintln(os.Stderr, "replay requires --recording /path/to/log.jsonl")
		return 2
	}

	entries, err := recording.Load(*recordingPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load recording: %v\n", err)
		return 1
	}

	server := recording.NewReplayServer(*port, entries)
	if *socket != "" {
		server.SetUnixSocket(*socket)
	}

	go func() {
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "replay server error: %v\n", err)
			os.Exit(1)
		}
	}()

	fmt.Printf("foxbridge replay serving %s from %s\n", server.ListenDescription(), *recordingPath)
	fmt.Printf("browser endpoint: %s\n", server.BrowserWSURL())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	return 0
}
