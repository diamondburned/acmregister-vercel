package handler

import (
	"net/http"

	"github.com/diamondburned/acmregister-vercel/internal/servutil"
	"github.com/diamondburned/acmregister/acmregister/bot"
	"github.com/diamondburned/acmregister/acmregister/env"
	"github.com/diamondburned/acmregister/acmregister/logger"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/pkg/errors"
)

func HandleInteraction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	botToken, err := env.BotToken()
	if err != nil {
		servutil.WriteErr(w, r, 500, err)
		return
	}

	opts, err := env.BotOpts(logger.Silent(ctx))
	if err != nil {
		servutil.WriteErr(w, r, 500, err)
		return
	}
	defer opts.Store.Close()

	s := state.NewAPIOnlyState(botToken, nil).WithContext(ctx)
	h := bot.NewHandler(s, opts)
	serverVars := env.InteractionServer()

	srv, err := webhook.NewInteractionServer(serverVars.PubKey, h)
	if err != nil {
		servutil.WriteErr(w, r, 500, errors.Wrap(err, "cannot create interaction server"))
		return
	}
	srv.ErrorFunc = servutil.WriteErr
	srv.ServeHTTP(w, r)
}
