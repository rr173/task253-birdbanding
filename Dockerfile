FROM docker.m.daocloud.io/library/golang:1.26.3-bookworm

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -trimpath -o /out/birdbanding ./cmd/birdbanding

FROM docker.m.daocloud.io/library/alpine:3.20

WORKDIR /data
COPY --from=0 /out/birdbanding /usr/local/bin/birdbanding

RUN adduser -D -u 10001 birdbanding
USER birdbanding

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/birdbanding"]
CMD ["--addr", ":8080"]
