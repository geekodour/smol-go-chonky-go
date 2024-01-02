-- name: GetCat :one
SELECT * FROM cats
WHERE cat_id = $1 LIMIT 1;

-- name: ListCats :many
SELECT * FROM cats
ORDER BY name;

-- name: AddCat :one
INSERT INTO cats (
  name, age, description
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: UpdateCat :exec
UPDATE cats
  set name = $2,
  age = $3,
  description = $4
WHERE cat_id = $1;

-- name: DeleteCat :exec
DELETE FROM cats
WHERE cat_id = $1;
