# Variables to be specified externally.
variable "registry" {
  default = "quay.io/tembo"
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
authors = "Tembo"
url = "https://github.com/tembo-io/temback"

target "default" {
  platforms = ["linux/amd64", "linux/arm64"]
  context = "."
  dockerfile-inline = <<EOT
  FROM alpine:3.21
  RUN apk update && apk add --no-cache ca-certificates postgresql-client
  ARG TARGETOS TARGETARCH
  COPY _build/$${TARGETOS}-$${TARGETARCH}/temback /usr/local/bin/temback
  ENTRYPOINT ["/usr/local/bin/temback"]
  CMD ["--version"]
  EOT
  tags = [
    "${registry}/temback:latest",
    "${registry}/temback:${version}",
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
