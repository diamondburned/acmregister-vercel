package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/diamondburned/acmregister/acmregister/bot"
	"github.com/diamondburned/acmregister/acmregister/env"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/listener"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	botToken := env.MustBotToken()

	opts, err := env.BotOpts(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	var start func()
	var h *bot.Handler

	defer func() {
		h.Close()
		opts.Store.Close()
	}()

	if server := env.InteractionServer(); server.Addr != "" {
		ses := state.NewAPIOnlyState(botToken, nil)

		h = bot.NewHandler(ses, opts)

		interactionServer, err := webhook.NewInteractionServer(server.PubKey, h)
		if err != nil {
			log.Fatalln("cannot create interaction server handler:", err)
		}

		httpServer := &http.Server{
			Addr:    server.Addr,
			Handler: interactionServer,
		}

		start = func() {
			log.Println("listening and serve HTTP at", httpServer.Addr)

			if err := listener.HTTPListenAndServeCtx(ctx, httpServer); err != nil {
				log.Fatalln("cannot serve HTTP:", err)
			}
		}
	} else {
		ses := state.New(botToken)
		ses.AddHandler(func(*gateway.ReadyEvent) {
			user, _ := ses.Me()
			log.Println("connected to Discord as", user.Tag())
		})
		defer ses.Close()

		h = bot.NewHandler(ses, opts)

		ses.AddIntents(h.Intents())
		ses.AddInteractionHandler(h)

		start = func() {
			log.Println("connecting to the Discord gateway...")

			if err := ses.Connect(ctx); err != nil {
				log.Fatalln("cannot connect:", err)
			}
		}
	}

	if err := h.OverwriteCommands(); err != nil {
		log.Fatalln("cannot apply commands:", err)
	}

	start()
	log.Println("shutting down...")
}
