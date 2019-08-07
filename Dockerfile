FROM golang:1.12
WORKDIR /src/
COPY . .
ENV PLATFORMS=linux
ENV ARCHITECTURES=amd64
RUN make release

# Artefactor relies on use of a docker daemon for archiving
FROM docker:18.06.1-ce-dind
RUN apk update && apk add bash openssh-client
COPY --from=0 /src/bin/artefactor_linux_amd64 \
              /usr/local/bin/artefactor
COPY add_private_key /usr/local/bin/
ENTRYPOINT artefactor
