package postgres_test

const (
	addFilmQuery      = `INSERT INTO folder_item (folder_id, film_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	checkViolation    = "23514"
	collectionColumns = `
		c.id, c.title, c.slug, c.description,
		(SELECT COALESCE(fl.external_url, '/static/' || fl.storage_key)
		FROM file AS fl WHERE fl.id = c.cover_file_id)
	`
	countCollectionFilmsQuery = `
	SELECT count(*) FROM collection_film
	WHERE collection_id = $1
	`
	countFilmsQuery = `
		SELECT count(*) FROM film`
	deleteFilmQuery = `
		DELETE FROM film
		WHERE id = $1
	`
	filmColumns = `
		f.id, f.title, f.original_title, f.film_type, f.production_year, f.duration_min, f.age_limit, f.description,
		(SELECT fl.external_url
			FROM film_file AS ff JOIN file AS fl ON fl.id=ff.file_id
			WHERE ff.film_id = f.id AND ff.file_role = 'poster'
			ORDER BY ff.sort_order LIMIT 1
		),
		(SELECT COALESCE(fl.external_url, '/static/' || fl.storage_key)
			FROM film_file AS ff JOIN file AS fl ON fl.id=ff.file_id
			WHERE ff.film_id = f.id AND ff.file_role = 'trailer'
			ORDER BY ff.sort_order LIMIT 1
		),
		(SELECT CASE WHEN fl.external_url IS NULL THEN fl.storage_key END
			FROM film_file AS ff JOIN file AS fl ON fl.id=ff.file_id
			WHERE ff.film_id = f.id AND ff.file_role = 'poster'
			ORDER BY ff.sort_order LIMIT 1
		)`
	filmGenresQuery = `
		SELECT fg.film_id, g.id, g.name, g.slug
		FROM film_genre AS fg
		JOIN genre AS g ON g.id = fg.genre_id
		WHERE fg.film_id = ANY($1)
		ORDER BY fg.film_id, g.id
	`
	folderQuery = `
		INSERT INTO folder (account_id, title, folder_type) VALUES ($1, $2, $3)
		ON CONFLICT (account_id, folder_type) WHERE folder_type <> 'custom'
		DO UPDATE SET folder_type = EXCLUDED.folder_type
		RETURNING id`
	foreignKeyViolation                   = "23503"
	getCollectionFilmRatingSummariesQuery = `
		WITH target_collection AS (
			SELECT id FROM collection WHERE id = $1
		),
		collection_total AS (
			SELECT count(*)::bigint AS total
			FROM collection_film
			WHERE collection_id = $1
		),
		page_films AS (
			SELECT cf.film_id, cf.position
			FROM collection_film AS cf
			JOIN target_collection AS c ON c.id = cf.collection_id
			ORDER BY cf.position
			LIMIT $2 OFFSET $3
		)
		SELECT c.id,
		       collection_total.total,
		       page_films.film_id,
		       COALESCE(ROUND(AVG(fr.score), 1), 0)::double precision,
		       count(fr.score)
		FROM target_collection AS c
		CROSS JOIN collection_total
		LEFT JOIN page_films ON TRUE
		LEFT JOIN film_rating AS fr ON fr.film_id = page_films.film_id
		GROUP BY c.id, collection_total.total, page_films.film_id, page_films.position
		ORDER BY page_films.position`
	getCollectionQuery = `
		SELECT ` + collectionColumns +
		` FROM collection AS c WHERE c.id = $1 `
	getFilmPosterQuery = `
 SELECT COALESCE(fl.storage_key, ''), fl.mime_type, COALESCE(fl.size_bytes, 0), fl.external_url
 FROM film AS f
 LEFT JOIN LATERAL (
  SELECT file.* FROM film_file AS ff JOIN file ON file.id = ff.file_id
  WHERE ff.film_id = f.id AND ff.file_role = 'poster'
  ORDER BY ff.sort_order LIMIT 1
 ) AS fl ON true
 WHERE f.id = $1`
	listCollectionFilmsQuery = `
		SELECT ` + filmColumns +
		` FROM collection_film AS cf
		  JOIN film AS f ON f.id = cf.film_id
		  WHERE cf.collection_id = $1
		  ORDER BY cf.position
		  LIMIT $2 OFFSET $3
		`
	listFilmsQuery = `
		SELECT ` + filmColumns +
		` FROM film AS f 
		ORDER BY f.id
		LIMIT $1 OFFSET $2`
	lockAccountQuery          = `SELECT id FROM account WHERE id = $1 FOR UPDATE`
	preferencesQuery          = `SELECT preferences FROM account WHERE id = $1 AND deleted_at IS NULL`
	removeFilmQuery           = `DELETE FROM folder_item WHERE folder_id = $1 AND film_id = $2`
	removeFromWatchLaterQuery = `
		DELETE FROM folder_item
		WHERE film_id = $2 AND folder_id IN (
			SELECT id FROM folder WHERE account_id = $1 AND folder_type = 'watch_later'
		)`
	savePreferenceQuery = `
		UPDATE account SET preferences = jsonb_set(preferences, ARRAY[$2]::text[], $3::jsonb, true)
		WHERE id = $1 AND deleted_at IS NULL`
	setFilmRatingQuery = `
		INSERT INTO film_rating (account_id, film_id, score)
		VALUES ($1, $2, $3)
		ON CONFLICT (account_id, film_id)
		DO UPDATE SET score = EXCLUDED.score
		RETURNING account_id, film_id, score, created_at, updated_at`
	updateFilmQuery = `
		UPDATE film SET title = $2, original_title = $3, film_type = $4, production_year = $5, duration_min = $6, age_limit = $7, description = $8
		WHERE id = $1
	`
)
