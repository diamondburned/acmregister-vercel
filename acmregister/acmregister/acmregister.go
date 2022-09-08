package acmregister

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/pkg/errors"
)

// ErrNotFound is returned if anything is not found.
var ErrNotFound = errors.New("not found")

// ErrMemberAlreadyExists is returned if a member is being registered with an
// existing member's information.
var ErrMemberAlreadyExists = errors.New("a member with your information already exists, contact the server administrator")

// ErrUnknownPronouns is returned if a Pronouns is unknown. Use NewPronouns and
// check its error to ensure that it is never invalid.
var ErrUnknownPronouns = errors.New("unknown pronouns")

type KnownGuild struct {
	GuildID           discord.GuildID
	ChannelID         discord.ChannelID
	RoleID            discord.RoleID
	InitUserID        discord.UserID
	RegisteredMessage string
}

type Member struct {
	GuildID  discord.GuildID
	UserID   discord.UserID
	Metadata MemberMetadata
}

type MemberMetadata struct {
	Email     Email    `json:"email"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Pronouns  Pronouns `json:"pronouns"`
}

// Name returns the first name and last if any.
func (m MemberMetadata) Name() string {
	name := m.FirstName
	if m.LastName != "" {
		name += " " + m.LastName
	}

	return name
}

// Nickname returns the nickname for the given member using their metadata.
func (m MemberMetadata) Nickname() string {
	nick := m.Name()

	switch m.Pronouns {
	case HiddenPronouns:
		// ok
	case AnyPronouns:
		nick += " (any pronouns)"
	default:
		nick += fmt.Sprintf(" (%s)", m.Pronouns)
	}

	return nick
}

// Pronouns describes a pronouns string in the format (they/them).
type Pronouns string

const (
	HeHim          Pronouns = "he/him"
	SheHer         Pronouns = "she/her"
	TheyThem       Pronouns = "they/them"
	AnyPronouns    Pronouns = "any"
	HiddenPronouns Pronouns = ""
)

var KnownPronouns = []Pronouns{
	HeHim,
	SheHer,
	TheyThem,
	AnyPronouns,
	HiddenPronouns,
}

// NewPronouns creates a new Pronouns value. An error is returned if the
// pronouns are invalid.
func NewPronouns(pronouns string) (Pronouns, error) {
	p := Pronouns(pronouns)
	return p, p.Validate()
}

func (p Pronouns) Validate() error {
	for _, known := range KnownPronouns {
		if known == p {
			return nil
		}
	}
	return ErrUnknownPronouns
}

// SubmissionSaveDuration is the duration to save submissions for.
const SubmissionSaveDuration = 1 * time.Hour

// ContainsContext can be embedded by any interface to have an overrideable
// context.
type ContainsContext interface {
	WithContext(context.Context) ContainsContext
}

// Store describes a Store instance. It combines all smaller stores.
type Store interface {
	io.Closer
	ContainsContext
	KnownGuildStore
	MemberStore
	SubmissionStore
}

// KnownGuildStore stores all known guilds, or guilds that are using the
// registration feature.
type KnownGuildStore interface {
	ContainsContext
	// InitGuild initializes a guild.
	InitGuild(KnownGuild) error
	// GuildInfo returns the information about a registered guild.
	GuildInfo(discord.GuildID) (*KnownGuild, error)
	// DeleteGuild deletes the guild with the given ID from the registered
	// database.
	DeleteGuild(discord.GuildID) error
}

// MemberStore stores all registered members.
type MemberStore interface {
	ContainsContext
	// MemberInfo returns the member's info for the given user.
	MemberInfo(discord.GuildID, discord.UserID) (*MemberMetadata, error)
	// RegisterMember registers the given member into the store.
	RegisterMember(Member) error
	// UnregisterMember unregisters the given member from the store.
	UnregisterMember(discord.GuildID, discord.UserID) error
}

// SubmissionStore stores submissions for a short while so that forms can be
// temproarily stored.
type SubmissionStore interface {
	ContainsContext
	// SaveSubmission temporarily saves a submission. The submission doesn't
	// have to be valid.
	SaveSubmission(Member) error
	// RestoreSubmission returns a saved submission.
	RestoreSubmission(discord.GuildID, discord.UserID) (*MemberMetadata, error)
}
