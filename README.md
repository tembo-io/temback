Temback
=======

A little Go app that takes a full backup of a Postgres cluster and uploads it
to S3.

To use, first authenticate to S3 with the appropriate profile. Then:

```sh
temback --help

env PGHOST=postgres.example.org PGUSER=postgres PGPASSWORD=XXXXXXX \
temback --name example-backup
```

This will create a directory named `example-backup` that contains:

*   `README.md`: A brief description of the backup, including the host name,
    timestamp, and Postgres version, plus brief instructions to restore.
*   `roles.sql`: A dump of all the database roles
*   `tablespaces.sql`: A dump of all the tablespaces
*   Directories starting with `db-` for each database, containing the output
    of the [`pg_dump`] directory format.

The `--text` option instead dumps each database as a plain text file named
`db-$dbname.sql`.

With `--compress` or `--bucket`, this directory will be archived as a tarball
named `example-backup.tar.gz`. The `--bucket` option uploads this tarball to
the specified S3 bucket.

Restore
-------

To restore this backup to a Postgres cluster running the same or later version
of Postgres as the original, change into the backup directory, then execute
these commands (assuming a superuser named `postgres`):

```sh
PGUSER=postgres
psql -f roles.sql
psql -f tablespaces.sql
pg_restore -j 8 -f db-postgres
pg_restore -C -j 8 -f db-app
```

Use `-C` to create a database before restoring it. Modify the `-j` option to
change the number of parallel jobs restoring each database.

For a backup made with `--text`, use `psql` to restore the databases:

```sh
for dir in db-*; do psql -f "$dir"; done
```

Plain text backups do not support parallel restores.

Options
-------

*   `--name`: The name of the backup; required
*   `--bucket`: Upload to the named S3 bucket
*   `--compress`: Compress into a tarball (ignored with `--bucket`)
*   `--host`: The Postgres host name; defaults to `PGHOST` if set
*   `--user`: The Postgres username; defaults to `PGUSER` if set
*   `--pass`: The Postgres password; defaults to `$PGPASSWORD` (preferred)
*   `--text`: Dump plain text mode; defaults to directory mode
*   `--clean`: Delete temporary files

Building
--------

To build the app, install Go and run:

```sh
make temback
```

To run it with the `--version` option:

```sh
make run
```

To see the location of the binary:

```sh
make show-build
```

Baking
------

To bake a docker image, start a docker registry and use `make`:

```sh
docker run -d -p 5001:5000 --restart=always --name registry registry:2
make image PUSH=true
```

Then pull it and run it:

```sh
docker pull localhost:5001/temback:latest
docker run --rm localhost:5001/temback:latest
```

[`pg_dump`]: https://www.postgresql.org/docs/current/app-pgdump.html
