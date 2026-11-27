package postgres

// The page and the standalone count query share this CTE. Counts cover both
// film types even when a page is restricted to a single type.
const filmSearchMatches = `
WITH query_words AS (
    SELECT word
    FROM regexp_split_to_table(lower($1), '[^[:alnum:]]+') AS words(word)
    WHERE word <> ''
), credit_people AS (
    SELECT p.id, btrim(p.first_name || ' ' || p.last_name) AS name
    FROM person AS p
    WHERE $4::boolean AND char_length($1) >= 3
      AND EXISTS (SELECT 1 FROM query_words)
      AND EXISTS (
          SELECT 1
          FROM (VALUES (p.first_name || ' ' || p.last_name), (p.original_name)) AS names(full_name)
          WHERE full_name IS NOT NULL AND NOT EXISTS (
              SELECT 1 FROM query_words AS q
              WHERE NOT EXISTS (
                  SELECT 1
                  FROM regexp_split_to_table(lower(names.full_name), '[^[:alnum:]]+') AS n(word)
                  WHERE starts_with(n.word, q.word)
              )
          )
      )
), credit_matches AS (
    SELECT fp.film_id, p.id, p.name, array_agg(DISTINCT rt.code ORDER BY rt.code) AS roles
    FROM credit_people AS p
    JOIN film_person AS fp ON fp.person_id = p.id
    JOIN role_type AS rt ON rt.id = fp.role_type_id AND rt.code IN ('actor', 'director')
    GROUP BY fp.film_id, p.id, p.name
), title_matches AS (
    SELECT f.id,
           CASE
               WHEN lower(f.title) = lower($1) OR lower(f.original_title) = lower($1) THEN 0
               WHEN lower(f.title) LIKE lower($2) ESCAPE '!' OR lower(f.original_title) LIKE lower($2) ESCAPE '!' THEN 1
               WHEN lower(f.title) LIKE lower($3) ESCAPE '!' OR lower(f.original_title) LIKE lower($3) ESCAPE '!' THEN 2
               ELSE 3
           END AS match_kind,
           GREATEST(word_similarity(lower($1), lower(f.title)),
                    word_similarity(lower($1), lower(f.original_title))) AS score
    FROM film AS f
    WHERE lower(f.title) LIKE lower($3) ESCAPE '!' OR lower(f.original_title) LIKE lower($3) ESCAPE '!'
       OR lower(f.title) % lower($1) OR lower(f.original_title) % lower($1)
       OR lower($1) <% lower(f.title) OR lower($1) <% lower(f.original_title)
), candidates AS (
    SELECT id, match_kind, score, true AS title_match FROM title_matches
    UNION ALL
    SELECT DISTINCT c.film_id, 4, 0::real, false
    FROM credit_matches AS c
    WHERE NOT EXISTS (SELECT 1 FROM title_matches AS t WHERE t.id = c.film_id)
), filtered AS (
    SELECT f.id, f.film_type, f.created_at, c.match_kind, c.score, c.title_match,
           rating.average_score, rating.votes, release.release_date, reviews.review_count
    FROM candidates AS c
    JOIN film AS f ON f.id = c.id
    LEFT JOIN LATERAL (
        SELECT round(avg(fr.score), 1)::double precision AS average_score, count(*) AS votes
        FROM film_rating AS fr
        WHERE fr.film_id = f.id AND ($18::boolean OR $15::double precision IS NOT NULL OR $16::double precision IS NOT NULL)
    ) AS rating ON true
    LEFT JOIN LATERAL (
        SELECT min(fr.release_date) AS release_date
        FROM film_release AS fr
        WHERE fr.film_id = f.id AND ($18 OR $13::date IS NOT NULL OR $14::date IS NOT NULL)
    ) AS release ON true
    LEFT JOIN LATERAL (
        SELECT count(*) AS review_count
        FROM review AS r
        WHERE $18 AND r.film_id = f.id AND r.deleted_at IS NULL
    ) AS reviews ON true
    WHERE ($11::integer IS NULL OR f.production_year >= $11)
      AND ($12::integer IS NULL OR f.production_year <= $12)
      AND ($13::date IS NULL OR release.release_date >= $13)
      AND ($14::date IS NULL OR release.release_date <= $14)
      AND ($15::double precision IS NULL OR rating.average_score >= $15)
      AND ($16::double precision IS NULL OR rating.average_score <= $16)
      AND (coalesce(cardinality($17::integer[]), 0) = 0 OR f.age_limit = ANY($17))
      AND NOT EXISTS (
          SELECT 1 FROM film_genre AS fg JOIN genre AS g ON g.id = fg.genre_id
          WHERE fg.film_id = f.id AND g.slug = ANY($6::text[])
      )
      AND (coalesce(cardinality($5::text[]), 0) = 0 OR (
          SELECT count(*)
          FROM film_genre AS fg JOIN genre AS g ON g.id = fg.genre_id
          WHERE fg.film_id = f.id AND g.slug = ANY($5)
      ) >= CASE WHEN $7::boolean THEN 1 ELSE cardinality($5) END)
      AND NOT EXISTS (
          SELECT 1 FROM film_country AS fc JOIN country AS co ON co.id = fc.country_id
          WHERE fc.film_id = f.id AND co.iso_code = ANY($9::text[])
      )
      AND (coalesce(cardinality($8::text[]), 0) = 0 OR (
          SELECT count(*)
          FROM film_country AS fc JOIN country AS co ON co.id = fc.country_id
          WHERE fc.film_id = f.id AND co.iso_code = ANY($8)
      ) >= CASE WHEN $10::boolean THEN 1 ELSE cardinality($8) END)
), totals AS (
    SELECT count(*) FILTER (WHERE film_type = 'movie') AS movies,
           count(*) FILTER (WHERE film_type = 'series') AS series
    FROM filtered
)
`

const filmSearchPage = `
, page AS (
    SELECT m.*, row_number() OVER (ORDER BY %s, m.id) AS position
    FROM filtered AS m
    WHERE $19::text = '' OR m.film_type = $19
    ORDER BY position
    LIMIT $20 OFFSET $21
)
SELECT totals.movies, totals.series,
       f.id, coalesce(f.title, ''), f.original_title, coalesce(f.film_type, ''),
       coalesce(f.production_year, 0), f.duration_min, coalesce(f.age_limit, 0), f.description,
       poster.external_url, poster.storage_key, trailer.url,
       coalesce(genres.value, '[]'::jsonb), coalesce(page.title_match, false),
       coalesce(credits.value, '[]'::jsonb),
       coalesce(page.average_score, 0), coalesce(page.votes, 0),
       coalesce(page.review_count, 0), page.release_date, coalesce(page.created_at, 'epoch'::timestamptz)
FROM totals
LEFT JOIN page ON true
LEFT JOIN film AS f ON f.id = page.id
LEFT JOIN LATERAL (
    SELECT fl.external_url, CASE WHEN fl.external_url IS NULL THEN fl.storage_key END AS storage_key
    FROM film_file AS ff JOIN file AS fl ON fl.id = ff.file_id
    WHERE ff.film_id = f.id AND ff.file_role = 'poster'
    ORDER BY ff.sort_order, ff.file_id LIMIT 1
) AS poster ON true
LEFT JOIN LATERAL (
    SELECT coalesce(fl.external_url, '/static/' || fl.storage_key) AS url
    FROM film_file AS ff JOIN file AS fl ON fl.id = ff.file_id
    WHERE $18 AND ff.film_id = f.id AND ff.file_role = 'trailer'
    ORDER BY ff.sort_order, ff.file_id LIMIT 1
) AS trailer ON true
LEFT JOIN LATERAL (
    SELECT jsonb_agg(jsonb_build_object('id', g.id, 'name', g.name, 'slug', g.slug) ORDER BY g.id) AS value
    FROM film_genre AS fg JOIN genre AS g ON g.id = fg.genre_id
    WHERE fg.film_id = f.id
) AS genres ON true
LEFT JOIN LATERAL (
    SELECT jsonb_agg(jsonb_build_object('id', c.id, 'name', c.name, 'roles', c.roles) ORDER BY c.id) AS value
    FROM credit_matches AS c
    WHERE c.film_id = f.id
) AS credits ON true
ORDER BY page.position
`
