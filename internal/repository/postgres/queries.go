package postgres

const (
	querySetTeamName = `insert into reviewer_service.teams (team_name) values ($1)`

	querySaveTeamMember = `insert into reviewer_service.users 
			(user_id, username, team_name, is_active) values ($1, $2, $3, $4)`

	queryTeamExists = `select exists (select 1 from reviewer_service.teams where team_name = $1)`
)
