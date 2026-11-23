# Salvanza for a NAS (Synology Container Manager, QNAP Container Station, or
# any Docker host): the background service runs the Backup Sets and their
# schedules, and the text window (salvanza-cli tui) is served to a web browser
# by ttyd, behind a login.
#
#   docker build -t salvanza-nas docker/nas
#   docker build --build-arg SALVANZA_REF=<commit or tag> -t salvanza-nas docker/nas
#
# Both programs are pure Go (CGO_ENABLED=0), built here from the sources at
# SALVANZA_REF, the same way packaging/debian/build-deb.sh builds them. The
# text window is newer than release 2.0.0, hence a commit of develop.

ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.20

FROM golang:${GO_VERSION}-alpine AS build
ARG SALVANZA_REF=4d737af0febfc7a8b2a7e79aee5ead4774ec4f60
ARG SALVANZA_REPO=https://github.com/Salvanza/salvanza_client
RUN apk add --no-cache curl tar
WORKDIR /src
RUN curl -fsSL "${SALVANZA_REPO}/archive/${SALVANZA_REF}.tar.gz" | tar -xz --strip-components=1
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
# go.work lists every module, cli included.
RUN go build -trimpath -ldflags "-s -w -X main.version=${SALVANZA_REF}" -o /out/salvanza-cli ./cli
RUN cd gui && go build -trimpath -tags service -ldflags "-s -w -X main.appVersion=${SALVANZA_REF}" -o /out/salvanza-svc .
RUN /out/salvanza-cli backup-folder -version

FROM alpine:${ALPINE_VERSION}
ARG SALVANZA_REF=4d737af0febfc7a8b2a7e79aee5ead4774ec4f60
# ttyd: the web terminal. tzdata: TZ for the schedules. ca-certificates: e-mail
# reports over TLS (PBS itself is pinned by its certificate fingerprint).
RUN apk add --no-cache ttyd tzdata ca-certificates
COPY --from=build /out/salvanza-cli /out/salvanza-svc /usr/local/bin/
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod 0755 /usr/local/bin/entrypoint.sh

LABEL org.opencontainers.image.title="salvanza-nas" \
      org.opencontainers.image.description="Salvanza service + web text window: NAS folder backups to Proxmox Backup Server" \
      org.opencontainers.image.source="https://github.com/Salvanza/salvanza_client" \
      org.opencontainers.image.licenses="GPL-3.0" \
      org.opencontainers.image.revision="${SALVANZA_REF}"

# The service's settings, Backup Sets, history and logs.
VOLUME ["/var/lib/salvanza"]
# The web text window.
EXPOSE 7681
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["serve"]
