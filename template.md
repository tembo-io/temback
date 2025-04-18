# {{.Name}} Backup

Host:     {{.Host}}
Date:     {{.Date}}
Postgres: {{.Version}}

This archive contains a backup of the complete contents of the PostgreSQL host
listed above.

To restore the complete cluster, execute these commands on a new server
running PostgreSQL {{.Version}} or later, assuming the superuser `postgres` and the
default database is `postgres`:

```sh
export PGUSER=postgres
pgsql -f roles.sql
pgsql -f tablespaces.sql
{{ range .Restores -}}
{{.}}
{{ end -}}
```

Change `PGUSER` and set other connection [environment variables] as
appropriate for your new server. Consult the [`psql`]{{ if eq .Format "dir" }} and [`pg_restore`]
documentation for details.{{else}} documentation for
details.{{end}}

[environment variables]: https://www.postgresql.org/docs/current/libpq-envars.html
[`psql`]: https://www.postgresql.org/docs/current/app-psql.html
{{ if eq .Format "dir" -}}
[`pg_restore`]: https://www.postgresql.org/docs/current/app-pgrestore.html
{{ end }}
