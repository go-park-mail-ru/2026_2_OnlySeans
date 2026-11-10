ALTER TABLE review
    ADD COLUMN version integer NOT NULL DEFAULT 1,
    ADD CONSTRAINT review_version_check CHECK (version > 0);

COMMENT ON COLUMN review.version IS
    'Номер версии строки для оптимистической блокировки: клиент присылает версию, которую читал, и UPDATE проходит только если она совпала.';
