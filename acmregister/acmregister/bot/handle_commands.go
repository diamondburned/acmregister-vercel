package bot

import (
	"encoding/json"
	"fmt"

	"github.com/diamondburned/acmregister/acmregister"
	"github.com/diamondburned/acmregister/acmregister/logger"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/pkg/errors"
)

var globalCommands = []api.CreateCommandData{
	{
		Name:        "init-register",
		Description: "initialize a channel to post a message",
		Options: []discord.CommandOption{
			&discord.ChannelOption{
				OptionName:  "channel",
				Description: "the channel to make a message in",
				Required:    true,
				ChannelTypes: []discord.ChannelType{
					discord.GuildText,
				},
			},
			&discord.RoleOption{
				OptionName:  "registered-role",
				Description: "the role to give when the user is registered",
				Required:    true,
			},
			&discord.StringOption{
				OptionName:  "message",
				Description: "body for the message",
				Required:    true,
			},
			&discord.StringOption{
				OptionName:  "register-button-label",
				Description: "the text for the Register button, default 'Register'",
			},
			&discord.StringOption{
				OptionName:  "registered-message",
				Description: "the message to reply once registered successfully, default \"You're all set!\"",
			},
		},
	},
	{
		Name:        "registered-member",
		Description: "group of commands that are member-related specific to this guild",
		Options: []discord.CommandOption{
			&discord.SubcommandOption{
				OptionName:  "query",
				Description: "query the info of a known user or error if not registered",
				Options: []discord.CommandOptionValue{
					&discord.UserOption{
						OptionName:  "who",
						Description: "the user to query",
						Required:    true,
					},
				},
			},
			&discord.SubcommandOption{
				OptionName:  "unregister",
				Description: "unregister a user and remove their role",
				Options: []discord.CommandOptionValue{
					&discord.UserOption{
						OptionName:  "who",
						Description: "the user to unregister",
						Required:    true,
					},
				},
			},
			&discord.SubcommandOption{
				OptionName:  "reset-name",
				Description: "change the name of a registered member back to the default one",
				Options: []discord.CommandOptionValue{
					&discord.UserOption{
						OptionName:  "who",
						Description: "the user to rename",
						Required:    true,
					},
				},
			},
		},
	},
	{
		Name:        "clear-registration",
		Description: "clear the Register message",
	},
}

// OverwriteCommands overwrites the commands to the ones defined in Commands.
func (h *Handler) OverwriteCommands() error {
	app, err := h.s.CurrentApplication()
	if err != nil {
		return errors.Wrap(err, "cannot get current app")
	}

	_, err = h.s.BulkOverwriteCommands(app.ID, globalCommands)
	if err != nil {
		return errors.Wrap(err, "cannot overwrite old commands")
	}

	return nil
}

func (h *Handler) cmdInit(ev *discord.InteractionEvent, opts discord.CommandInteractionOptions) *api.InteractionResponse {
	_, err := h.store.GuildInfo(ev.GuildID)
	if err == nil {
		return errorResponse(errors.New("guild is already registered; clear it first"))
	}

	var data struct {
		ChannelID             discord.ChannelID `discord:"channel"`
		RegisteredRole        discord.RoleID    `discord:"registered-role"`
		Message               string            `discord:"message"`
		RegisteredButtonLabel string            `discord:"register-button-label?"`
		RegisteredMessage     string            `discord:"registered-message?"`
	}

	if err := opts.Unmarshal(&data); err != nil {
		return errorResponse(err)
	}

	if data.RegisteredButtonLabel == "" {
		data.RegisteredButtonLabel = registeredButtonLabel
	}

	if data.RegisteredMessage == "" {
		data.RegisteredMessage = registeredMessage
	}

	registerMsg, err := h.s.SendMessageComplex(data.ChannelID, api.SendMessageData{
		Content: data.Message,
		Components: []discord.ContainerComponent{
			&discord.ActionRowComponent{
				&discord.ButtonComponent{
					Style:    discord.PrimaryButtonStyle(),
					CustomID: "register",
					Label:    data.RegisteredButtonLabel,
				},
			},
		},
	})
	if err != nil {
		return errorResponse(errors.Wrap(err, "cannot send register message"))
	}

	if err := h.store.InitGuild(acmregister.KnownGuild{
		GuildID:           ev.GuildID,
		ChannelID:         data.ChannelID,
		RoleID:            data.RegisteredRole,
		InitUserID:        ev.SenderID(),
		RegisteredMessage: data.RegisteredMessage,
	}); err != nil {
		h.s.DeleteMessage(data.ChannelID, registerMsg.ID, "cannot init guild, check error")
		return errorResponse(errors.Wrap(err, "cannot init guild"))
	}

	return msgResponse(&api.InteractionResponseData{
		Flags:   api.EphemeralResponse,
		Content: option.NewNullableString("Done!"),
	})
}

func (h *Handler) cmdMemberQuery(ev *discord.InteractionEvent, opts discord.CommandInteractionOptions) *api.InteractionResponse {
	guild, err := h.store.GuildInfo(ev.GuildID)
	if err != nil {
		logger := logger.FromContext(h.ctx)
		logger.Println("ignoring unknown guild", ev.GuildID)
		return nil
	}

	var data struct {
		Who discord.UserID `discord:"who"`
	}

	if err := opts.Unmarshal(&data); err != nil {
		return errorResponse(err)
	}

	metadata, err := h.store.MemberInfo(guild.GuildID, data.Who)
	if err != nil {
		return errorResponse(err)
	}

	b, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return errorResponse(errors.Wrap(err, "cannot encode metadata as JSON"))
	}

	return msgResponse(&api.InteractionResponseData{
		Content: option.NewNullableString("```json\n" + string(b) + "\n```"),
	})
}

func (h *Handler) cmdMemberUnregister(ev *discord.InteractionEvent, opts discord.CommandInteractionOptions) *api.InteractionResponse {
	guild, err := h.store.GuildInfo(ev.GuildID)
	if err != nil {
		logger := logger.FromContext(h.ctx)
		logger.Println("ignoring unknown guild", ev.GuildID)
		return nil
	}

	var data struct {
		Who discord.UserID `discord:"who"`
	}

	if err := opts.Unmarshal(&data); err != nil {
		return errorResponse(err)
	}

	target, err := h.s.Member(ev.GuildID, data.Who)
	if err != nil {
		return errorResponse(errors.Wrap(err, "invalid member for 'who'"))
	}

	if err := h.store.UnregisterMember(ev.GuildID, data.Who); err != nil {
		if errors.Is(err, acmregister.ErrNotFound) {
			err = errors.New("user is not registered")
		}
		return errorResponse(err)
	}

	if err := h.s.RemoveRole(
		ev.GuildID, data.Who, guild.RoleID,
		api.AuditLogReason(fmt.Sprintf(
			"%s requested for %s (%v) to be unregistered",
			ev.Sender().Tag(), target.User.Tag(), data.Who,
		)),
	); err != nil {
		return errorResponse(errors.Wrap(err, "cannot remove role, but member is registered"))
	}

	return msgResponse(&api.InteractionResponseData{
		Content:         option.NewNullableString("User " + data.Who.Mention() + " has been unregistered."),
		AllowedMentions: &api.AllowedMentions{},
	})
}

func (h *Handler) cmdMemberResetName(ev *discord.InteractionEvent, opts discord.CommandInteractionOptions) *api.InteractionResponse {
	guild, err := h.store.GuildInfo(ev.GuildID)
	if err != nil {
		logger := logger.FromContext(h.ctx)
		logger.Println("ignoring unknown guild", ev.GuildID)
		return nil
	}

	var data struct {
		Who discord.UserID `discord:"who"`
	}

	if err := opts.Unmarshal(&data); err != nil {
		return errorResponse(err)
	}

	metadata, err := h.store.MemberInfo(guild.GuildID, data.Who)
	if err != nil {
		return errorResponse(err)
	}

	if err := h.s.ModifyMember(ev.GuildID, data.Who, api.ModifyMemberData{
		Nick: option.NewString(metadata.Nickname()),
	}); err != nil {
		return errorResponse(err)
	}

	return msgResponse(&api.InteractionResponseData{
		Content:         option.NewNullableString("User " + data.Who.Mention() + "'s nickname has been reset."),
		AllowedMentions: &api.AllowedMentions{},
	})
}

func (h *Handler) cmdClear(ev *discord.InteractionEvent, opts discord.CommandInteractionOptions) *api.InteractionResponse {
	_, err := h.store.GuildInfo(ev.GuildID)
	if err != nil {
		logger := logger.FromContext(h.ctx)
		logger.Println("ignoring unknown guild", ev.GuildID)
		return nil
	}

	if err := h.store.DeleteGuild(ev.GuildID); err != nil {
		return errorResponse(err)
	}

	return msgResponse(&api.InteractionResponseData{
		Content: option.NewNullableString("Done. All members have been removed from the database, but their roles stay."),
	})
}
