# syntax=docker/dockerfile:1.1.7-experimental
FROM --platform=linux/amd64 golang:1.18-alpine AS build

ARG PKG_NAME
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN apk add --no-cache make git

WORKDIR /src
COPY go.mod ./
COPY go.sum ./

RUN --mount=type=cache,id=go-build-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/root/.cache/go-build \
	--mount=type=cache,id=go-pkg-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/go/pkg \
		go mod download -x

COPY . ./

RUN --mount=type=cache,id=go-build-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/root/.cache/go-build \
	--mount=type=cache,id=go-pkg-${TARGETOS}-${TARGETARCH}${TARGETVARIANT},target=/go/pkg \
		make build
RUN mv bin/${PKG_NAME}* /bin/

FROM scratch AS release-linux

ARG PKG_NAME
ARG VCS_REF
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

LABEL org.opencontainers.image.revision=$VCS_REF \
	org.opencontainers.image.source="https://github.com/hairyhenderson/${PKG_NAME}"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /bin/${PKG_NAME}_${TARGETOS}-${TARGETARCH}${TARGETVARIANT} /${PKG_NAME}

ENTRYPOINT [ "/${PKG_NAME}" ]

FROM alpine:3.15 AS alpine

ARG PKG_NAME
ARG VCS_REF
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

LABEL org.opencontainers.image.revision=$VCS_REF \
	org.opencontainers.image.source="https://github.com/hairyhenderson/${PKG_NAME}"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /bin/${PKG_NAME}_${TARGETOS}-${TARGETARCH}${TARGETVARIANT} /${PKG_NAME}

ENTRYPOINT [ "/bin/${PKG_NAME}" ]

FROM --platform=windows/amd64 mcr.microsoft.com/windows/nanoserver:1809 AS release-windows

ARG PKG_NAME
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
COPY --from=build /bin/${PKG_NAME}_${TARGETOS}-${TARGETARCH}${TARGETVARIANT}.exe /${PKG_NAME}.exe

FROM release-$TARGETOS AS release
