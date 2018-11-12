FROM golang:1.10
WORKDIR /go/src/github.com/appvia/artefactor/
COPY . .
ENV PLATFORMS=linux
ENV ARCHITECTURES=amd64
RUN make release

# Artefactor relies on use of a docker daemon for archiving
FROM docker:18.06.1-ce-dind
RUN apk update && apk add bash
COPY --from=0 /go/src/github.com/appvia/artefactor/bin/artefactor_linux_amd64 /usr/sbin/artefactor
ENTRYPOINT artefactor
