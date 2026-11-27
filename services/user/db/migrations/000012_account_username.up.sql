-- Правила имени пользователя приводятся к правилам бэкенда и фронтенда:
-- буквы любого алфавита, цифры, пробел, дефис и подчёркивание, от 2 до 32 символов.
-- Набор символов проверяет бэкенд: классы символов в регулярных выражениях
-- PostgreSQL зависят от локали базы, поэтому в ограничении остаются длина
-- и отсутствие пробелов по краям.
-- Удаление и добавление в одной команде: если новое ограничение не встанет,
-- старое не пропадёт.
ALTER TABLE account
    DROP CONSTRAINT account_username_check,
    ADD CONSTRAINT account_username_check
        CHECK (length(username) BETWEEN 2 AND 32 AND username = btrim(username));

COMMENT ON CONSTRAINT account_username_check ON account IS
    'Имя от 2 до 32 символов без пробелов по краям. Допустимые символы проверяет бэкенд.';
