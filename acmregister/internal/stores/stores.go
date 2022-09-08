package stores

import (
	"io"
	"log"

	"github.com/diamondburned/acmregister/acmregister"
	"github.com/diamondburned/acmregister/acmregister/verifyemail"
)

type StoreCloser interface {
	acmregister.Store
	verifyemail.PINStore
	io.Closer
}

func Must[T acmregister.Store](store T, err error) T {
	if err != nil {
		log.Fatalln("cannot make store:", err)
	}
	return store
}
