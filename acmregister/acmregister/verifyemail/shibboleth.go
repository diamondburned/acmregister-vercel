package verifyemail

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/diamondburned/acmregister/acmregister"
	"github.com/pkg/errors"
)

// ShibbolethVerifier implements VerifyEmail.
type ShibbolethVerifier struct {
	// URL is a Shibboleth URL that redirects to the SSO portal.
	URL string
}

// VerifyEmail implements acmregister.EmailVerifier.
func (v ShibbolethVerifier) Verify(ctx context.Context, email acmregister.Email) error {
	username := email.Username()
	if username == "" {
		return errors.New("invalid email")
	}

	jar, _ := cookiejar.New(nil)
	client := shibbolethClient{
		Client: &http.Client{
			Jar: jar,
			Transport: wrapTransport(nil, func(r *http.Request, rt http.RoundTripper) (*http.Response, error) {
				r.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:102.0) Gecko/20100101 Firefox/102.0")
				r.Header.Set("Origin", "https://shibboleth.fullerton.edu")
				return rt.RoundTrip(r)
			}),
		},
		ctx: ctx,
	}

	shibbolethURL, err := client.followRedirect(v.URL)
	if err != nil {
		return errors.Wrap(err, "cannot follow redirect")
	}

	errorReq, err := client.PostForm(shibbolethURL.String(), url.Values{
		"j_username":       {username},
		"j_password":       {"a"},
		"_eventId_proceed": {""},
	})
	if err != nil {
		return errors.Wrap(err, "cannot test logging in")
	}
	defer errorReq.Body.Close()

	errorDoc, err := goquery.NewDocumentFromReader(errorReq.Body)
	if err != nil {
		return errors.Wrap(err, "cannot parse logging in HTML body")
	}
	errorReq.Body.Close()

	loginErrorElem := errorDoc.Find(`form[name="loginForm"] p.form-error`).First()
	if loginErrorElem == nil {
		return errors.Wrap(err, "cannot find p.form-error")
	}

	switch loginErr := strings.TrimSpace(loginErrorElem.Text()); loginErr {
	case "The password you entered was incorrect.":
		return nil
	case "The username you entered cannot be identified.":
		return errors.New("your email is not in the CSU Fullerton registry")
	default:
		return fmt.Errorf("unknown login error: %q", loginErr)
	}
}

type shibbolethClient struct {
	*http.Client
	ctx context.Context
}

func (c *shibbolethClient) followRedirect(uri string) (*url.URL, error) {
	r, err := c.Get(uri)
	if err != nil {
		return nil, err
	}
	r.Body.Close()
	return r.Request.URL, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func wrapTransport(old http.RoundTripper, f func(*http.Request, http.RoundTripper) (*http.Response, error)) http.RoundTripper {
	if old == nil {
		old = http.DefaultTransport
	}
	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return f(r, old)
	})
}
