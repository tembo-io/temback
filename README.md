Shutdown Backup
===============

A little Go app that takes a full backup of a Postgres cluster and uploads it
to S3.

To use, first authenticate to S3 with the appropriate profile. Then:

```sh
go run . --help

env PGHOST=postgres.example.org \
    PGUSER=postgres \
    PGPASSWORD=XXXXXXX \
go run . --name org_xyz-inst_abc-my_db --bucket my-backup-database
```

This will create a directory named `org_xyz-inst_abc-my_db` that contains:

*   `roles.sql`: A dump of all the database roles
*   `tablespaces.sql`: A dump of all the tablespaces
*   Directories starting with `db-` for each database, containing the output
    of the [`pg_dump`] directory format.

The `--text` option instead dumps each database as a plain text file named
`db-$dbname.sql`.

This directory will be archived as a tarball named
`org_xyz-inst_abc-my_db.tar.gz` that's uploaded to the S3 bucket specified by
`--bucket`.

Restore
-------

To restore from this backup to a Postgres cluster running the same or later
version of Postgres as the original, decompress the archive and change into
the backup directory, then execute these commands (assuming a superuser named
`postgres`):

```sh
PGUSER=postgres
psql -f roles.sql
psql -f tablespaces.sql
for dir in db-*; do pg_restore -j 8 -f "$dir"; done
```

Modify the `-j` option to change the number of parallel jobs restoring each
database.

For a backup made with `--text`, change that last line to:

```sh
for dir in db-*; do psql -f "$dir"; done
```

Plain text backups do not support parallel restores.

Options
-------

*   `--name`: The name of the backup
*   `--host`: The Postgres host name; defaults to `PGHOST` if set
*   `--user`: The Postgres username; defaults to `PGUSER` if set
*   `--pass`: The Postgres password; defaults to `$PGPASSWORD` (preferred)
*   `--bucket`: The S3 bucket name
*   `--text`: Dump plain text

[`pg_dump`]: https://www.postgresql.org/docs/current/app-pgdump.html
