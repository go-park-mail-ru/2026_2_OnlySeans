DROP TABLE IF EXISTS review;
DROP TRIGGER IF EXISTS film_rating_write_history ON film_rating;
DROP FUNCTION IF EXISTS write_film_rating_history();
DROP TABLE IF EXISTS film_rating_history;
DROP TABLE IF EXISTS film_rating;
