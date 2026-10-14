-- name: SetOrgScope :exec
-- Transaction-local organization scope read by row-level-security policies (ORG-04).
SELECT set_config('app.org_id', sqlc.arg(org_id)::text, true);

-- name: CurrentOrgScope :one
SELECT coalesce(current_setting('app.org_id', true), '')::text AS org_id;
