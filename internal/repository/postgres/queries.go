package postgres

const (
	querySetTeamName = `insert into reviewer_service.teams (team_name) values ($1)`

	querySaveTeamMember = `insert into reviewer_service.users 
			(user_id, username, team_name, is_active) values ($1, $2, $3, $4)`

	queryGetTeam = `select user_id, username, is_active from reviewer_service.users where team_name=$1`

	queryTeamExists = `select exists (select 1 from reviewer_service.teams where team_name = $1)`
)
