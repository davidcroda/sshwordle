package sshwordle

import (
	"context"
	"fmt"
	"github.com/charmbracelet/wish"
	lm "github.com/charmbracelet/wish/logging"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gliderlabs/ssh"

	bm "github.com/charmbracelet/wish/bubbletea"
)

func StartServer(host string, port int, useApi bool) {
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithPublicKeyAuth(publicKeyHandler),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			bm.Middleware(getTeaHandler(useApi)),
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Starting SSH server on %s:%d", host, port)
	go func() {
		if err = s.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-done
	log.Println("Stopping SSH server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}

func publicKeyHandler(_ctx ssh.Context, _key ssh.PublicKey) bool {
	return true
}

func getTeaHandler(useApi bool) bm.BubbleTeaHandler {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		pty, _, active := s.Pty()
		if !active {
			fmt.Println("no active terminal, skipping")
			return nil, nil
		}

		rand.Seed(time.Now().UnixNano())

		var backend Backend
		if useApi {
			backend = NewApiBackend()
		} else {
			backend = NewStaticBackend()
		}

		g := NewGame(pty.Window.Width, pty.Window.Height, s, backend)
		return g, nil
	}
}
