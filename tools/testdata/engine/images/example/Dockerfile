ARG BASE_IMAGE
FROM ghcr.io/databasifylabs/engine-sdk:1.0.0@sha256:4444444444444444444444444444444444444444444444444444444444444444 AS sdk

FROM ${BASE_IMAGE}
COPY --from=sdk /databasify /usr/local/bin/databasify
