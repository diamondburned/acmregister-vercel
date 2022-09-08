-- Language: postgresql
--
CREATE TABLE
	meta (v SMALLINT NOT NULL);

INSERT INTO
	meta(v)
VALUES
	(1);

-- NEW VERSION
UPDATE
	meta
SET
	v = 2;

CREATE TABLE
	known_guilds (
		guild_id BIGINT PRIMARY KEY,
		channel_id BIGINT NOT NULL,
		role_id BIGINT NOT NULL,
		init_user_id BIGINT NOT NULL,
		registered_message TEXT NOT NULL
	);

CREATE TABLE
	members (
		guild_id BIGINT NOT NULL REFERENCES known_guilds(guild_id) ON DELETE CASCADE,
		user_id BIGINT NOT NULL,
		email TEXT NOT NULL,
		metadata JSONB NOT NULL,
		UNIQUE (guild_id, user_id),
		UNIQUE (guild_id, email)
	);

CREATE TABLE
	registration_submissions (
		guild_id BIGINT NOT NULL REFERENCES known_guilds(guild_id) ON DELETE CASCADE,
		user_id BIGINT NOT NULL,
		metadata JSONB NOT NULL,
		expire_at TIMESTAMP NOT NULL,
		UNIQUE(guild_id, user_id)
	);

CREATE TABLE
	pin_codes (
		guild_id BIGINT NOT NULL REFERENCES known_guilds(guild_id) ON DELETE CASCADE,
		user_id BIGINT NOT NULL,
		pin SMALLINT NOT NULL,
		UNIQUE(guild_id, user_id),
		UNIQUE(guild_id, pin),
		FOREIGN KEY (guild_id, user_id) REFERENCES registration_submissions(guild_id, user_id) ON DELETE CASCADE
	);
