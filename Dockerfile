FROM ubuntu:bionic as builder

RUN apt update && apt install golang ca-certificates -y --no-install-recommends

ADD . .

WORKDIR /

RUN go build -o p2k ./cmd/p2k/main.go

FROM ubuntu:bionic

RUN apt update \
    && apt-get install -y wget python xz-utils xdg-utils \
    pandoc imagemagick \
    ca-certificates \
    && apt clean

RUN wget -nv -O- https://download.calibre-ebook.com/linux-installer.py | python -c "import sys; main=lambda:sys.stderr.write('Download failed\n'); exec(sys.stdin.read()); main()"

COPY --from=builder /p2k /usr/bin

ENTRYPOINT p2k
