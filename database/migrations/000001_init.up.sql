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

create table if not exists reviewer_service.pull_requests(
    pull_request_id text primary key,
    pull_request_name text not null,
    author_id text references reviewer_service.users,
    status text not null,
    assigned_reviewers text[] not null,
    created_at timestamptz,
    merged_at timestamptz
);
