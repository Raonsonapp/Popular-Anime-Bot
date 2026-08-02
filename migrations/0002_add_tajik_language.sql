-- Add Tajik as a supported UI language, and make it the default (the bot
-- itself now defaults new users to Tajik instead of Persian).

ALTER TABLE users DROP CONSTRAINT users_language_pref_check;
ALTER TABLE users ADD CONSTRAINT users_language_pref_check
    CHECK (language_pref IN ('tg', 'fa', 'ru', 'en'));
ALTER TABLE users ALTER COLUMN language_pref SET DEFAULT 'tg';

INSERT INTO languages (code, name) VALUES ('tg', 'Тоҷикӣ')
ON CONFLICT DO NOTHING;
