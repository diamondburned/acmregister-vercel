package bot

import (
	"context"
	"strings"
	"time"

	"github.com/diamondburned/acmregister/acmregister"
	"github.com/diamondburned/acmregister/acmregister/logger"
	"github.com/diamondburned/acmregister/acmregister/verifyemail"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/pkg/errors"
)

func (h *Handler) modalRegisterResponse(ev *discord.InteractionEvent, modal *discord.ModalInteraction) *api.InteractionResponse {
	guild, err := h.store.GuildInfo(ev.GuildID)
	if err != nil {
		logger := logger.FromContext(h.ctx)
		logger.Println("ignoring unknown guild", ev.GuildID)
		return nil
	}

	if _, err := h.store.MemberInfo(ev.GuildID, ev.SenderID()); err == nil {
		return errorResponse(errors.New("you're already registered!"))
	}

	var data struct {
		Email     acmregister.Email    `discord:"email"`
		FirstName string               `discord:"first"`
		LastName  string               `discord:"last?"`
		Pronouns  acmregister.Pronouns `discord:"pronouns?"`
	}

	if err := modal.Components.Unmarshal(&data); err != nil {
		return errorResponse(err)
	}

	metadata := acmregister.MemberMetadata(data)
	metadata.Email = acmregister.Email(strings.TrimSpace(string(metadata.Email)))
	metadata.FirstName = strings.TrimSpace(metadata.FirstName)
	metadata.LastName = strings.TrimSpace(metadata.LastName)

	member := acmregister.Member{
		GuildID:  ev.GuildID,
		UserID:   ev.SenderID(),
		Metadata: metadata,
	}

	if err := h.store.SaveSubmission(member); err != nil {
		h.logErr(ev.GuildID, errors.Wrap(err, "cannot save registration submission (not important)"))
		// not important so we continue
	}

	if err := metadata.Pronouns.Validate(); err != nil {
		return errorResponse(err)
	}

	if err := h.opts.verifyEmail(h.ctx, metadata.Email); err != nil {
		return errorResponse(err)
	}

	if h.opts.SMTPVerifier == nil {
		return h.registerAndRespond(ev, guild, metadata)
	}

	// This might take a while.
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()

		ctx, cancel := context.WithTimeout(h.ctx, 25*time.Second)
		defer cancel()

		if err := h.opts.SMTPVerifier.SendConfirmationEmail(ctx, member); err != nil {
			h.privateWarning(ev, errors.Wrap(err, "cannot send confirmation email"))
			h.followUp(ev, internalErrorResponse().Data)
			return
		}

		h.followUp(ev, &api.InteractionResponseData{
			Flags:   discord.EphemeralMessage,
			Content: option.NewNullableString(verifyPINMessage),
			Components: &discord.ContainerComponents{
				&discord.ActionRowComponent{
					&discord.ButtonComponent{
						Style:    discord.PrimaryButtonStyle(),
						CustomID: "verify-pin",
						Label:    verifyPINButtonLabel,
					},
				},
			},
		})
	}()

	return deferResponse(discord.EphemeralMessage)
}

func (h *Handler) modalVerifyPIN(ev *discord.InteractionEvent, modal *discord.ModalInteraction) *api.InteractionResponse {
	guild, err := h.store.GuildInfo(ev.GuildID)
	if err != nil {
		logger := logger.FromContext(h.ctx)
		logger.Println("ignoring unknown guild", ev.GuildID)
		return nil
	}

	if h.opts.SMTPVerifier == nil {
		metadata, err := h.store.RestoreSubmission(ev.GuildID, ev.SenderID())
		if err != nil {
			return errorResponse(errors.Wrap(err, "cannot restore your registration, try registering again"))
		}

		// Just in case the user manually triggered this interaction when this
		// feature is disabled. We don't want to crash the whole bot.
		return h.registerAndRespond(ev, guild, *metadata)
	}

	var data struct {
		PIN verifyemail.PIN `discord:"pin"`
	}

	if err := modal.Components.Unmarshal(&data); err != nil {
		return errorResponse(err)
	}

	metadata, err := h.opts.SMTPVerifier.ValidatePIN(ev.GuildID, ev.SenderID(), data.PIN)
	if err != nil {
		// Warn about weird errors just in case.
		if err != nil && !errors.Is(err, acmregister.ErrNotFound) {
			h.privateWarning(ev, errors.Wrap(err, "cannot validate PIN"))
		}

		return errorResponse(errors.New("incorrect PIN code given, try again"))
	}

	// At this point, the user ID matches with the known email, and the given
	// PIN also matches that email, so we're good.
	return h.registerAndRespond(ev, guild, *metadata)
}

func (h *Handler) registerAndRespond(ev *discord.InteractionEvent, guild *acmregister.KnownGuild, metadata acmregister.MemberMetadata) *api.InteractionResponse {
	member := acmregister.Member{
		GuildID:  ev.GuildID,
		UserID:   ev.SenderID(),
		Metadata: metadata,
	}

	if err := h.s.AddRole(ev.GuildID, ev.SenderID(), guild.RoleID, api.AddRoleData{
		AuditLogReason: "member registered, added by acmRegister",
	}); err != nil {
		h.privateWarning(ev, errors.Wrap(err, "cannot add role"))
		return internalErrorResponse()
	}

	if err := h.store.RegisterMember(member); err != nil {
		if errors.Is(err, acmregister.ErrMemberAlreadyExists) {
			return errorResponse(acmregister.ErrMemberAlreadyExists)
		} else {
			h.privateWarning(ev, errors.Wrap(err, "cannot save into database"))
			return internalErrorResponse()
		}
	}

	if err := h.s.ModifyMember(ev.GuildID, ev.SenderID(), api.ModifyMemberData{
		Nick: option.NewString(metadata.Nickname()),
	}); err != nil {
		h.privateWarning(ev, errors.Wrap(err, "cannot nickname new member (not important)"))
	}

	msg := guild.RegisteredMessage
	if msg == "" {
		msg = registeredMessage
	}

	return msgResponse(&api.InteractionResponseData{
		Flags:   discord.EphemeralMessage,
		Content: option.NewNullableString(msg),
	})
}
