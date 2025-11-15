create schema if not exists reviewer_service;

create table if not exists reviewer_service.teams(
    team_name text primary key
);

create table if not exists reviewer_service.users(
    user_id text primary key,
    username text not null,
    team_name text references reviewer_service.teams(team_name) on delete cascade,
    is_active boolean not null
);
