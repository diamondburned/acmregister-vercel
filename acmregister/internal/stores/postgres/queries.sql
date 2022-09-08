-- Language: postgresql
--
-- name: Version :one
SELECT
	v
FROM
	meta;

-- name: GuildInfo :one
SELECT
	*
FROM
	known_guilds
WHERE
	guild_id = $1
LIMIT
	1;

-- name: DeleteGuild :execrows
DELETE FROM
	known_guilds
WHERE
	guild_id = $1;

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
	($1, $2, $3, $4, $5);

-- name: RegisterMember :exec
INSERT INTO
	members (guild_id, user_id, email, metadata)
VALUES
	($1, $2, $3, $4);

-- name: UnregisterMember :execrows
DELETE FROM
	members
WHERE
	guild_id = $1
	AND user_id = $2;

-- name: MemberInfo :one
SELECT
	metadata
FROM
	members
WHERE
	guild_id = $1
	AND user_id = $2;

-- name: SaveSubmission :exec
INSERT INTO
	registration_submissions (guild_id, user_id, metadata, expire_at)
VALUES
	($1, $2, $3, NOW() + INTERVAL '1 hour') ON CONFLICT (guild_id, user_id)
DO
UPDATE
SET
	metadata = EXCLUDED.metadata,
	expire_at = EXCLUDED.expire_at;

-- name: DeleteSubmission :exec
DELETE FROM
	registration_submissions
WHERE
	guild_id = $1
	AND user_id = $2;

-- name: RestoreSubmission :one
SELECT
	metadata
FROM
	registration_submissions
WHERE
	guild_id = $1
	AND user_id = $2
	AND expire_at >= NOW();

-- name: CleanupSubmissions :exec
DELETE FROM
	registration_submissions
WHERE
	expire_at < NOW();

-- name: InsertPIN :exec
INSERT INTO
	pin_codes (guild_id, user_id, pin)
VALUES
	($1, $2, $3) ON CONFLICT (guild_id, user_id)
DO
UPDATE
SET
	pin = $3;

-- name: ValidatePIN :one
SELECT
	registration_submissions.metadata
FROM
	registration_submissions
	JOIN pin_codes ON registration_submissions.guild_id = pin_codes.guild_id
	AND registration_submissions.user_id = pin_codes.user_id
WHERE
	pin_codes.guild_id = $1
	AND pin_codes.user_id = $2
	AND pin_codes.pin = $3
	AND registration_submissions.expire_at >= NOW();
