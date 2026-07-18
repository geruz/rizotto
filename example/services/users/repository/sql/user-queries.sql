-- name: GetAllUsers :many
SELECT *
FROM public.users
ORDER BY created_at DESC;

-- name: GetUser :one
SELECT *
FROM public.users
WHERE id = $1;
