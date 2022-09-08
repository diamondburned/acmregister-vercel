package stores

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/diamondburned/acmregister/acmregister"
	"github.com/diamondburned/acmregister/acmregister/verifyemail"
	"github.com/diamondburned/acmregister/internal/stores/sqlite"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/pkg/errors"
)

type sqliteStore struct {
	q   *sqlite.Queries
	db  *sql.DB
	ctx context.Context
}

// NewSQLite creates a new SQLite store.
func NewSQLite(ctx context.Context, uri string) (StoreCloser, error) {
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, errors.Wrap(err, "sql/sqlite3")
	}

	if err := sqlite.Migrate(ctx, db); err != nil {
		return nil, errors.Wrap(err, "cannot migrate sqlite db")
	}

	return sqliteStore{
		q:   sqlite.New(db),
		db:  db,
		ctx: ctx,
	}, nil
}

func (s sqliteStore) Close() error {
	return s.db.Close()
}

func (s sqliteStore) WithContext(ctx context.Context) acmregister.ContainsContext {
	s.ctx = ctx
	return s
}

func (s sqliteStore) InitGuild(guild acmregister.KnownGuild) error {
	return s.q.InitGuild(s.ctx, sqlite.InitGuildParams{
		GuildID:           int64(guild.GuildID),
		ChannelID:         int64(guild.ChannelID),
		RoleID:            int64(guild.RoleID),
		InitUserID:        int64(guild.InitUserID),
		RegisteredMessage: guild.RegisteredMessage,
	})
}

func (s sqliteStore) GuildInfo(guildID discord.GuildID) (*acmregister.KnownGuild, error) {
	v, err := s.q.GuildInfo(s.ctx, int64(guildID))
	if err != nil {
		return nil, sqliteErr(err)
	}

	return &acmregister.KnownGuild{
		GuildID:           discord.GuildID(v.GuildID),
		ChannelID:         discord.ChannelID(v.ChannelID),
		RoleID:            discord.RoleID(v.RoleID),
		InitUserID:        discord.UserID(v.InitUserID),
		RegisteredMessage: v.RegisteredMessage,
	}, nil
}

func (s sqliteStore) DeleteGuild(guildID discord.GuildID) error {
	n, err := s.q.DeleteGuild(s.ctx, int64(guildID))
	if err != nil {
		return sqliteErr(err)
	}
	if n == 0 {
		return acmregister.ErrNotFound
	}
	return nil
}

func (s sqliteStore) MemberInfo(guildID discord.GuildID, userID discord.UserID) (*acmregister.MemberMetadata, error) {
	v, err := s.q.MemberInfo(s.ctx, sqlite.MemberInfoParams{
		GuildID: int64(guildID),
		UserID:  int64(userID),
	})
	if err != nil {
		return nil, sqliteErr(err)
	}

	var metadata acmregister.MemberMetadata

	if err := json.Unmarshal([]byte(v.Metadata), &metadata); err != nil {
		return nil, errors.Wrap(err, "member metadata JSON is corrupted")
	}

	return &metadata, nil
}

func (s sqliteStore) RegisterMember(m acmregister.Member) error {
	b, err := json.Marshal(m.Metadata)
	if err != nil {
		return errors.Wrap(err, "cannot encode member metadata as JSON")
	}

	if err := s.q.RegisterMember(s.ctx, sqlite.RegisterMemberParams{
		GuildID:  int64(m.GuildID),
		UserID:   int64(m.UserID),
		Email:    string(m.Metadata.Email),
		Metadata: string(b),
	}); err != nil {
		if sqlite.IsConstraintFailed(err) {
			return acmregister.ErrMemberAlreadyExists
		}
		return sqliteErr(err)
	}

	s.q.DeleteSubmission(s.ctx, sqlite.DeleteSubmissionParams{
		GuildID: int64(m.GuildID),
		UserID:  int64(m.UserID),
	})

	return nil
}

func (s sqliteStore) UnregisterMember(guildID discord.GuildID, userID discord.UserID) error {
	n, err := s.q.UnregisterMember(s.ctx, sqlite.UnregisterMemberParams{
		GuildID: int64(guildID),
		UserID:  int64(userID),
	})
	if err != nil {
		return sqliteErr(err)
	}
	if n == 0 {
		return acmregister.ErrNotFound
	}
	return nil
}

func (s sqliteStore) SaveSubmission(m acmregister.Member) error {
	b, err := json.Marshal(m.Metadata)
	if err != nil {
		return errors.Wrap(err, "cannot encode member metadata as JSON")
	}

	err = s.q.SaveSubmission(s.ctx, sqlite.SaveSubmissionParams{
		GuildID:  int64(m.GuildID),
		UserID:   int64(m.UserID),
		Metadata: string(b),
	})
	if err != nil {
		return sqliteErr(err)
	}

	s.q.CleanupSubmissions(s.ctx)
	return nil
}

func (s sqliteStore) RestoreSubmission(guildID discord.GuildID, userID discord.UserID) (*acmregister.MemberMetadata, error) {
	b, err := s.q.RestoreSubmission(s.ctx, sqlite.RestoreSubmissionParams{
		GuildID: int64(guildID),
		UserID:  int64(userID),
	})
	if err != nil {
		return nil, sqliteErr(err)
	}

	var metadata acmregister.MemberMetadata

	if err := json.Unmarshal([]byte(b), &metadata); err != nil {
		return nil, errors.Wrap(err, "member metadata JSON is corrupted")
	}

	return &metadata, nil
}

func (s sqliteStore) GeneratePIN(guildID discord.GuildID, userID discord.UserID) (verifyemail.PIN, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return verifyemail.InvalidPIN, ctx.Err()
		default:
			pin := verifyemail.GeneratePIN()
			err := s.q.InsertPIN(ctx, sqlite.InsertPINParams{
				GuildID: int64(guildID),
				UserID:  int64(userID),
				Pin:     int64(pin),
			})
			if err == nil {
				return pin, nil
			}
			if sqlite.IsConstraintFailed(err) {
				continue
			}
			return verifyemail.InvalidPIN, errors.Wrap(sqliteErr(err), "cannot store PIN")
		}
	}
}

func (s sqliteStore) ValidatePIN(guildID discord.GuildID, userID discord.UserID, pin verifyemail.PIN) (*acmregister.MemberMetadata, error) {
	b, err := s.q.ValidatePIN(s.ctx, sqlite.ValidatePINParams{
		GuildID:   int64(guildID),
		GuildID_2: int64(guildID),
		UserID:    int64(userID),
		Pin:       int64(pin),
	})
	if err != nil {
		return nil, sqliteErr(err)
	}

	var metadata acmregister.MemberMetadata

	if err := json.Unmarshal([]byte(b), &metadata); err != nil {
		return nil, errors.Wrap(err, "member metadata JSON is corrupted")
	}

	return &metadata, nil
}

func sqliteErr(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return acmregister.ErrNotFound
	}
	return err
}
