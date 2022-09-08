// Package env contains environment variable bindings to acmregister settings.
package env

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/diamondburned/acmregister/acmregister"
	"github.com/diamondburned/acmregister/acmregister/bot"
	"github.com/diamondburned/acmregister/acmregister/logger"
	"github.com/diamondburned/acmregister/acmregister/verifyemail"
	"github.com/diamondburned/acmregister/internal/stores"
)

// BotOpts gets bot.Opts from the environment variables.
func BotOpts(ctx context.Context) (bot.Opts, error) {
	logger := logger.FromContext(ctx)

	var store stores.StoreCloser

	switch driver := os.Getenv("STORE_DRIVER"); driver {
	case "sqlite":
		store = stores.Must(stores.NewSQLite(ctx, os.Getenv("SQLITE_URL")))
		logger.Println("using SQLite")
	case "postgresql":
		store = stores.Must(stores.NewPostgreSQL(ctx, os.Getenv("POSTGRESQL_URL")))
		logger.Println("using PostgreSQL")
	default:
		logger.Fatalf("unknown $STORE_DRIVER %q", driver)
	}

	opts := bot.Opts{
		Store: store,
		EmailHosts: acmregister.EmailHostsVerifier{
			"csu.fullerton.edu",
			"fullerton.edu",
		},
	}

	if shibbolethURL := os.Getenv("VERIFY_SHIBBOLETH_URL"); shibbolethURL != "" {
		log.Println("enabling Shibboleth verifier")
		opts.ShibbolethVerifier = &verifyemail.ShibbolethVerifier{
			URL: shibbolethURL,
		}
	}

	smtpInfo := verifyemail.SMTPInfo{
		Host:         os.Getenv("VERIFY_SMTP_HOST"),
		Email:        os.Getenv("VERIFY_SMTP_EMAIL"),
		Password:     os.Getenv("VERIFY_SMTP_PASSWORD"),
		TemplatePath: os.Getenv("VERIFY_SMTP_TEMPLATE_PATH"),
	}

	if smtpInfo != (verifyemail.SMTPInfo{}) {
		logger.Println("got SMTP credentials, enabling SMTP verification")
		v, err := verifyemail.NewSMTPVerifier(smtpInfo, store)
		if err != nil {
			logger.Fatalln("cannot create SMTP verifier:", err)
		}

		opts.SMTPVerifier = v
	}

	return opts, nil
}

type InteractionServerVars struct {
	Addr   string // $INTERACTION_SERVER_ADDRESS
	PubKey string // $INTERACTION_SERVER_PUBKEY
}

// InteractionServer gets $INTERACTION_SERVER_* variables.
func InteractionServer() InteractionServerVars {
	return InteractionServerVars{
		Addr:   os.Getenv("INTERACTION_SERVER_ADDRESS"),
		PubKey: os.Getenv("INTERACTION_SERVER_PUBKEY"),
	}
}

// MustBotToken exits if there's no $BOT_TOKEN.
func MustBotToken() string {
	t, err := BotToken()
	if err != nil {
		log.Fatalln(err)
	}
	return t
}

// BotToken gets $BOT_TOKEN with the Bot prefix.
func BotToken() (string, error) {
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		return "", errors.New("missing $BOT_TOKEN")
	}

	if !strings.HasPrefix(botToken, "Bot ") {
		botToken = "Bot " + botToken
	}

	return botToken, nil
}
