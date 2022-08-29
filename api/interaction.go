package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/diamondburned/acmregister-vercel/internal/servutil"
	"github.com/diamondburned/acmregister/acmregister/bot"
	"github.com/diamondburned/acmregister/acmregister/env"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/pkg/errors"
)

var interactionHandler = func() http.Handler {
	botToken, err := env.BotToken()
	if err != nil {
		log.Fatalln("cannot get bot token:", err)
	}

	opts, err := env.BotOpts(context.Background())
	if err != nil {
		log.Fatalln("cannot get bot opts:", err)
	}

	serverVars := env.InteractionServer()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := state.NewAPIOnlyState(botToken, nil).WithContext(r.Context())
		h := bot.NewHandler(s, opts)

		srv, err := webhook.NewInteractionServer(serverVars.PubKey, h)
		if err != nil {
			servutil.WriteErr(w, r, 500, errors.Wrap(err, "cannot create interaction server"))
			return
		}
		srv.ErrorFunc = servutil.WriteErr
		srv.ServeHTTP(w, r)
	})
}()

func HandleInteraction(w http.ResponseWriter, r *http.Request) {
	interactionHandler.ServeHTTP(w, r)
}
