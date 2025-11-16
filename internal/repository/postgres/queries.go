package postgres

const (
	querySetTeamName = `insert into reviewer_service.teams (team_name) values ($1)`

	querySaveTeamMember = `insert into reviewer_service.users 
    		(user_id, username, team_name, is_active) values ($1, $2, $3, $4)`

	queryGetTeam = `select user_id, username, is_active from reviewer_service.users where team_name = $1`

	querySetIsActive = `update reviewer_service.users set is_active = $2 
    		where user_id = $1 returning user_id, username, team_name, is_active`

	querySavePR = `insert into reviewer_service.pull_requests
    		(pull_request_id, pull_request_name, author_id, status, assigned_reviewers, created_at)
			values ($1, $2, $3, $4, $5, $6)`

	querySetPRStatus = `update reviewer_service.pull_requests
			set status = $2, merged_at = coalesce(merged_at, $3) where pull_request_id = $1
    		returning pull_request_id, pull_request_name, author_id, status, assigned_reviewers, created_at, merged_at`

	queryTeamExists = `select exists (select 1 from reviewer_service.teams where team_name = $1)`

	queryPRExists = `select exists (select 1 from reviewer_service.pull_requests where pull_request_id = $1)`

	queryGetTeamName = `select team_name from reviewer_service.users where user_id = $1`

	queryGetActiveReviewers = `select user_id from reviewer_service.users
    		where team_name = $1 and is_active = true and user_id <> $2 order by random() limit 2`
)
