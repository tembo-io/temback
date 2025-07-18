# Variables to be specified externally.
variable "registry" {
  default = "ghcr.io/theory"
  description = "The image registry."
}

variable "version" {
  default = ""
  description = "The release version."
}

variable "revision" {
  default = ""
  description = "The current Git commit SHA."
}

# Values to use in the targets.
now = timestamp()
authors = "David E. Wheeler"
url = "https://github.com/theory/temback"

target "default" {
  platforms = ["linux/amd64", "linux/arm64"]
  matrix = {
    pgv = ["17", "16", "15", "14"]
  }
  name = "temback-${pgv}"
  context = "."
  dockerfile-inline = <<EOT
  FROM ubuntu:24.04
  ADD https://salsa.debian.org/postgresql/postgresql-common/-/raw/master/pgdg/apt.postgresql.org.sh .
  RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && chmod +x apt.postgresql.org.sh && ./apt.postgresql.org.sh -p -y && rm apt.postgresql.org.sh && apt-get install -y --no-install-recommends postgresql-client-${pgv} && apt-get clean && rm -rf /var/cache/apt/* /var/lib/apt/lists/* /usr/share/postgresql/${pgv}/man
  ARG TARGETOS TARGETARCH
  COPY _build/$${TARGETOS}-$${TARGETARCH}/temback /usr/local/bin/temback
  ENTRYPOINT ["/usr/local/bin/temback"]
  CMD ["--version"]
  EOT
  tags = [
    "${registry}/temback:latest-pg${pgv}",
    "${registry}/temback:${version}-pg${pgv}",
  ]
  annotations = [
    "index,manifest:org.opencontainers.image.created=${now}",
    "index,manifest:org.opencontainers.image.url=${url}",
    "index,manifest:org.opencontainers.image.source=${url}",
    "index,manifest:org.opencontainers.image.version=${version}",
    "index,manifest:org.opencontainers.image.revision=${revision}",
    "index,manifest:org.opencontainers.image.vendor=${authors}",
    "index,manifest:org.opencontainers.image.title=Temback",
    "index,manifest:org.opencontainers.image.description=Temback PostgreSQL Backup to S3",
    "index,manifest:org.opencontainers.image.documentation=${url}",
    "index,manifest:org.opencontainers.image.authors=${authors}",
    "index,manifest:org.opencontainers.image.licenses=PostgreSQL",
    "index,manifest:org.opencontainers.image.base.name=scratch",
  ]
  labels = {
    "org.opencontainers.image.created" = "${now}",
    "org.opencontainers.image.url" = "${url}",
    "org.opencontainers.image.source" = "${url}",
    "org.opencontainers.image.version" = "${version}",
    "org.opencontainers.image.revision" = "${revision}",
    "org.opencontainers.image.vendor" = "${authors}",
    "org.opencontainers.image.title" = "Temback",
    "org.opencontainers.image.description" = "Temback PostgreSQL Backup to S3",
    "org.opencontainers.image.documentation" = "${url}",
    "org.opencontainers.image.authors" = "${authors}",
    "org.opencontainers.image.licenses" = "PostgreSQL"
    "org.opencontainers.image.base.name" = "scratch",
  }
}
