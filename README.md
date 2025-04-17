Shutdown Backup
===============

A little Go app that takes a full backup of a Postgres cluster and uploads it
to S3.

```sh
go run . --help

env PGHOST=postgres.example.org \
    PGUSER=postgres \
    PGPASSWORD=XXXXXXX \
go run . --name org_xyz-inst_abc-my_db --bucket my-backup-database
```

Options
-------

*   `--name`: The name of the backup
*   `--host`: The Postgres host name
*   `--user`: The Postgres username
*   `--pass`: The Postgres password 
*   `--bucket`: The S3 bucket name
*   `--text`: Dump plain text
