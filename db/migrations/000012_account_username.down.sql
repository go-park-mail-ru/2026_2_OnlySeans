-- Откат не пройдёт, если в базе уже есть имена с кириллицей, заглавными буквами,
-- пробелами, дефисами или короче 3 символов: их нужно переименовать заранее.
ALTER TABLE account
    DROP CONSTRAINT account_username_check,
    ADD CONSTRAINT account_username_check
        CHECK (username ~ '^[a-z0-9_]{3,32}$');
