FROM golang:alpine AS builder

RUN apk add upx ca-certificates tzdata

WORKDIR /usr/src

COPY . /usr/src

RUN go mod download
RUN gover=$(go version | awk '{print $3,$4}') \
    && today=$(date '+%y%m%d') \
    && [ -n "$GITHUB_SHA" ] && stage=" (git-${GITHUB_SHA:0:7})" || stage="" \
    && version="$today$stage ($gover)" \
    && sed -i "s#YOURVERSION#$version#g" cmd/root.go \
    && CGO_ENABLED=0 go build -ldflags="-s -w -extldflags='static'" -trimpath -o app \
    && upx --best --lzma app

FROM scratch AS prod

COPY --from=builder /usr/share/zoneinfo/UTC /etc/localtime
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/src/app /socks5ssh

ENTRYPOINT ["/socks5ssh"]
