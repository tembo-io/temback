# Changelog

All notable changes to this project will be documented in this file. It uses the
[Keep a Changelog] format, and this project adheres to [Semantic Versioning].

  [Keep a Changelog]: https://keepachangelog.com/en/1.1.0/
  [Semantic Versioning]: https://semver.org/spec/v2.0.0.html
    "Semantic Versioning 2.0.0"

## [v0.1.0] — To Be Released

### ⚡ Improvements

*   First release, everything is new!
*   Full database backup following the [depesz backup pattern]
*   Uses [`pg_dumpall`] to dump global objects and  [`pg_dump`] to dump each
    database
*   Supports parallel directory and plain text dumps
*   Generates a `README.md` to guide restoration
*   Uploads resulting backup tarball to S3

### 🏗️ Build Setup

*   Built with Go
*   Compiled for a number of platforms
*   Download the binary from [GitHub]

### 📚 Documentation

*   Build and install docs in the [README]

  [v0.1.0]: https://github.com/tembo-io/temback/compare/feec925...v0.1.0
  [depesz backup pattern]: https://www.depesz.com/2019/12/10/how-to-effectively-dump-postgresql-databases/
  [`pg_dump`]: https://www.postgresql.org/docs/current/app-pgdump.html
  [`pg_dumpall`]: https://www.postgresql.org/docs/current/app-pg-dumpall.html
  [GitHub]: https://github.com/tembo-io/temback/releases
  [README]: https://github.com/tembo-io/temback/blob/v0.1.0/README.md
