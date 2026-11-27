package persons

const personSearchMatches = `
WITH name_matches AS (
    SELECT p.id, btrim(p.first_name || ' ' || p.last_name) AS name, p.original_name,
           p.photo_file_id, p.created_at,
           CASE
               WHEN lower(p.first_name || ' ' || p.last_name) = lower($1) OR lower(p.original_name) = lower($1) THEN 0
               WHEN lower(p.first_name || ' ' || p.last_name) LIKE lower($2) ESCAPE '!'
                 OR lower(p.original_name) LIKE lower($2) ESCAPE '!' THEN 1
               WHEN lower(p.first_name || ' ' || p.last_name) LIKE lower($3) ESCAPE '!'
                 OR lower(p.original_name) LIKE lower($3) ESCAPE '!' THEN 2
               ELSE 3
           END AS match_kind,
           GREATEST(word_similarity(lower($1), lower(p.first_name || ' ' || p.last_name)),
                    word_similarity(lower($1), lower(p.original_name))) AS score
    FROM person AS p
    WHERE lower(p.first_name || ' ' || p.last_name) LIKE lower($3) ESCAPE '!'
       OR lower(p.original_name) LIKE lower($3) ESCAPE '!'
       OR lower(p.first_name || ' ' || p.last_name) % lower($1)
       OR lower(p.original_name) % lower($1)
       OR lower($1) <% lower(p.first_name || ' ' || p.last_name)
       OR lower($1) <% lower(p.original_name)
), filtered AS (
    SELECT m.*, roles.codes AS role_codes
    FROM name_matches AS m
    CROSS JOIN LATERAL (
        SELECT array_agg(DISTINCT rt.code ORDER BY rt.code) AS codes
        FROM film_person AS fp
        JOIN role_type AS rt ON rt.id = fp.role_type_id AND rt.code IN ('actor', 'director')
        WHERE fp.person_id = m.id
    ) AS roles
    WHERE roles.codes IS NOT NULL
      AND NOT coalesce(roles.codes && $5::text[], false)
      AND (coalesce(cardinality($4::text[]), 0) = 0 OR
           CASE WHEN $6::boolean THEN roles.codes && $4 ELSE roles.codes @> $4 END)
), totals AS (
    SELECT count(*) AS total FROM filtered
)
`

const personSearchPage = `
, page AS (
    SELECT m.*, row_number() OVER (ORDER BY %s, m.id) AS position
    FROM filtered AS m
    ORDER BY position
    LIMIT $7 OFFSET $8
)
SELECT totals.total, page.id, coalesce(page.name, ''), coalesce(page.original_name, ''),
       coalesce(photo.external_url, '/static/' || photo.storage_key),
       coalesce(to_jsonb(page.role_codes), '[]'::jsonb),
       CASE WHEN $9::boolean AND page.id IS NOT NULL THEN (
           SELECT count(DISTINCT fp.film_id)
           FROM film_person AS fp
           JOIN role_type AS rt ON rt.id = fp.role_type_id AND rt.code IN ('actor', 'director')
           WHERE fp.person_id = page.id
       ) END AS works_count
FROM totals
LEFT JOIN page ON true
LEFT JOIN file AS photo ON photo.id = page.photo_file_id
ORDER BY page.position
`
