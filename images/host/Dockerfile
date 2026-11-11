ARG GO_IMAGE=golang:1.27.3-trixie@sha256:af1bfc2614ebb1dbd31af514b78740fd678850e76498762b425ecee4c5c6ea80

FROM ${GO_IMAGE} AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /databasify ./cmd/databasify-host

FROM scratch
ARG SDK_VERSION
LABEL io.databasify.sdk=$SDK_VERSION \
      org.opencontainers.image.source=https://github.com/databasifylabs/engine-sdk
COPY --from=build /databasify /databasify
ENTRYPOINT ["/databasify"]
