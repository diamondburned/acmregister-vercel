package handler

import (
	"net/http"

	"github.com/diamondburned/acmregister-vercel/internal/servutil"
	"github.com/diamondburned/acmregister/acmregister/bot"
	"github.com/diamondburned/acmregister/acmregister/env"
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

	opts, err := env.BotOpts(ctx)
	if err != nil {
		servutil.WriteErr(w, r, 500, err)
		return
	}

	s := state.NewAPIOnlyState(botToken, nil).WithContext(ctx)
	h := bot.NewHandler(s, opts)
	serverVars := env.InteractionServer()

	srv, err := webhook.NewInteractionServer(serverVars.PubKey, h)
	if err != nil {
		opts.Store.Close()

		servutil.WriteErr(w, r, 500, errors.Wrap(err, "cannot create interaction server"))
		return
	}
	srv.ErrorFunc = servutil.WriteErr
	srv.ServeHTTP(w, r)

	// Wait for the interaction to die off before closing. We can't defer
	// close any of this, because we have to spawn a goroutine for a
	// follow-up response, so we want to wait until that's done before we
	// clean up.
	//
	// Vercel, of course, behaves in a different way than you would a main()
	// program, in that any dangling goroutines will actually prevent the
	// whole serverless function, so this method works. If not for that,
	// then I don't really know what else to do.
	h.Wait()
	h.Close()
	opts.Store.Close()
}
