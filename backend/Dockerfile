FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/oncall-api ./cmd/oncall-api

FROM alpine:3.21

RUN adduser -D -H oncall
USER oncall
COPY --from=build /out/oncall-api /usr/local/bin/oncall-api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/oncall-api"]
