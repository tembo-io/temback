# {{.Name}} Backup

This backup was made for PostgreSQL host {{.Host}} on {{.Date}}.
Execute these commands on a new server running PostgreSQL {{.Version}} or later,
assuming the superuser `postgres`:

```sh
export PGUSER=postgres
pgsql -f roles.sql
pgsql -f tablespaces.sql

```
