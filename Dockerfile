FROM debian:sid-slim as builder

RUN apt update && apt install golang ca-certificates -y --no-install-recommends

ADD . .

WORKDIR /

RUN go build -o p2k ./cmd/p2k/main.go

FROM debian:sid-slim

RUN apt update && apt install calibre ca-certificates pandoc imagemagick -y --no-install-recommends && apt clean

COPY --from=builder /p2k /usr/bin

ENTRYPOINT p2k
