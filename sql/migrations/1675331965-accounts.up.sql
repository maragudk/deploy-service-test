create table accounts (
  id text primary key default ('a_' || lower(hex(randomblob(16)))),
  created text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  updated text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  name text not null
) strict;

create trigger accounts_updated_timestamp after update on accounts begin
  update accounts set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where id = old.id;
end;

create table groups (
  id text primary key default ('g_' || lower(hex(randomblob(16)))),
  created text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  updated text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  accountID text not null references accounts (id) on delete cascade,
  name text not null
) strict;

create trigger groups_updated_timestamp after update on groups begin
  update groups set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where id = old.id;
end;

create table users (
  id text primary key default ('u_' || lower(hex(randomblob(16)))),
  created text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  updated text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  accountID text not null references accounts (id) on delete cascade,
  name text not null,
  email text unique not null,
  confirmed int not null default 0,
  active int not null default 1
) strict;

create trigger users_updated_timestamp after update on users begin
  update users set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where id = old.id;
end;

create table group_membership (
  groupID text not null references groups (id) on delete cascade,
  userID text not null references users (id) on delete cascade,
  primary key (groupID, userID)
) strict;

create table tokens (
  value text primary key,
  created text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  updated text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  userID text not null references users (id) on delete cascade,
  used int not null default 0,
  expires text not null default (strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '7 days'))
) strict;

create trigger tokens_updated_timestamp after update on tokens begin
  update tokens set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where value = old.value;
end;
