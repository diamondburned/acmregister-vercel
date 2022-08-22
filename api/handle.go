package api

import (
	"encoding/json"
	"net/http"

	"github.com/diamondburned/acmregister/acmregister/bot"
	"github.com/diamondburned/acmregister/acmregister/env"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/pkg/errors"
)

func Handle(w http.ResponseWriter, r *http.Request) {
	botToken, err := env.BotToken()
	if err != nil {
		writeErr(w, 500, err)
		return
	}

	opts, err := env.BotOpts(r.Context())
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	defer opts.Store.Close()

	s := state.NewAPIOnlyState(botToken, nil)
	h := bot.NewHandler(s, opts)
	serverVars := env.InteractionServer()

	srv, err := webhook.NewInteractionServer(serverVars.PubKey, h)
	if err != nil {
		writeErr(w, 500, errors.Wrap(err, "cannot create interaction server"))
		return
	}

	srv.ServeHTTP(w, r)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	var errBody struct {
		Error string `json:"error"`
	}

	if err != nil {
		errBody.Error = err.Error()
	} else {
		errBody.Error = http.StatusText(code)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errBody)
}
