FROM golang:1.27.4-alpine3.24 AS build

WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
RUN GOBIN=/out go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1

COPY cmd ./cmd
COPY internal ./internal
RUN go build -trimpath -buildvcs=false -o /out/rydo-api ./cmd/rydo-api

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/rydo-api /app/rydo-api
COPY --from=build /out/migrate /usr/local/bin/migrate
COPY db/migrations /app/db/migrations
COPY --chmod=755 scripts/railway-migrate.sh /app/railway-migrate

USER 65532:65532
CMD ["/app/rydo-api"]
