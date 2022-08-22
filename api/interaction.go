package handler

import (
	"encoding/json"
	"log"
	"net/http"

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
		writeErr(w, r, 500, err)
		return
	}

	opts, err := env.BotOpts(logger.Silent(ctx))
	if err != nil {
		writeErr(w, r, 500, err)
		return
	}
	defer opts.Store.Close()

	s := state.NewAPIOnlyState(botToken, nil).WithContext(ctx)
	h := bot.NewHandler(s, opts)
	serverVars := env.InteractionServer()

	srv, err := webhook.NewInteractionServer(serverVars.PubKey, h)
	if err != nil {
		writeErr(w, r, 500, errors.Wrap(err, "cannot create interaction server"))
		return
	}
	srv.ErrorFunc = writeErr
	srv.ServeHTTP(w, r)
}

func writeErr(w http.ResponseWriter, _ *http.Request, code int, err error) {
	var errBody struct {
		Error string `json:"error"`
	}

	if err != nil {
		errBody.Error = err.Error()
		log.Println("request error:", err)
	} else {
		errBody.Error = http.StatusText(code)
		log.Println("request responded with status", code)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errBody)
}