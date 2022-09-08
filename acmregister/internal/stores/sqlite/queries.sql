-- Language: sqlite
--
-- name: GuildInfo :one
SELECT
	*
FROM
	known_guilds
WHERE
	guild_id = ?
LIMIT
	1;

-- name: DeleteGuild :execrows
DELETE FROM
	known_guilds
WHERE
	guild_id = ?;

-- name: InitGuild :exec
INSERT INTO
	known_guilds (
		guild_id,
		channel_id,
		init_user_id,
		role_id,
		registered_message
	)
VALUES
	(?, ?, ?, ?, ?);

-- name: RegisterMember :exec
INSERT INTO
	members (guild_id, user_id, email, metadata)
VALUES
	(?, ?, ?, ?);

-- name: UnregisterMember :execrows
DELETE FROM
	members
WHERE
	guild_id = ?
	AND user_id = ?;

-- name: MemberInfo :one
SELECT
	*
FROM
	members
WHERE
	guild_id = ?
	AND user_id = ?;

-- name: SaveSubmission :exec
REPLACE INTO registration_submissions (guild_id, user_id, metadata, expire_at)
VALUES
	(?, ?, ?, UNIXEPOCH('now', '+1 hours'));

-- name: DeleteSubmission :exec
DELETE FROM
	registration_submissions
WHERE
	guild_id = ?
	AND user_id = ?;

-- name: RestoreSubmission :one
SELECT
	metadata
FROM
	registration_submissions
WHERE
	guild_id = ?
	AND user_id = ?
	AND expire_at >= UNIXEPOCH();

-- name: CleanupSubmissions :exec
DELETE FROM
	registration_submissions
WHERE
	expire_at < UNIXEPOCH();

-- name: InsertPIN :exec
INSERT INTO
	pin_codes (guild_id, user_id, pin)
VALUES
	(?, ?, ?) ON CONFLICT (guild_id, user_id) DO
UPDATE
SET
	pin = EXCLUDED.pin;

-- name: ValidatePIN :one
SELECT
	metadata
FROM
	registration_submissions
WHERE
	registration_submissions.guild_id = ?
	AND registration_submissions.user_id = (
		SELECT
			user_id
		FROM
			pin_codes
		WHERE
			pin_codes.guild_id = ?
			AND pin_codes.user_id = ?
			AND pin_codes.pin = ?
	)
	AND registration_submissions.expire_at >= UNIXEPOCH();
