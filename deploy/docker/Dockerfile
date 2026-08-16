# Container build for earlcameron.com — the AFF-style replacement for building on the box.
#
# THE BUILD CONTEXT IS THE WORKSPACE ROOT, NOT THIS REPO. go.mod pins GoWebComponents and
# CashFlux by relative replace (../GoWebComponents, ../CashFlux), so the context must hold
# all three checkouts as siblings — the same layout CI assembles and the droplet used to
# keep under /opt/earlcameron/src:
#
#   <context>/PersonalWebsiteMid2026/   this repo
#   <context>/GoWebComponents/          at the GWC_REF pin from deploy/lib.sh (submodules!)
#   <context>/CashFlux/                 at the CASHFLUX_REF pin from deploy/lib.sh
#
# The release workflow (.github/workflows/release.yml) checks the three out exactly as
# ci.yml does and builds with `-f PersonalWebsiteMid2026/deploy/docker/Dockerfile .`.
#
# What this buys over the legacy on-box build: the droplet stops needing a Go toolchain,
# 2 GB of build headroom and a swapfile — it just pulls an image. What stays the same: the
# runtime contract (LISTEN_ADDR, DB_PATH, web/static + web/cashflux resolved relative to
# the working directory) is untouched, so the same binary layout serves either way.

FROM golang:1.26-bookworm AS build

# The three siblings, laid out as the replace directives expect.
WORKDIR /src
COPY GoWebComponents/ GoWebComponents/
COPY CashFlux/ CashFlux/
COPY PersonalWebsiteMid2026/ PersonalWebsiteMid2026/

WORKDIR /src/PersonalWebsiteMid2026

# Server + this site's own wasm client + wasm_exec.js. CGO off is not a preference: the
# runtime image is distroless/static and a cgo binary will not run there (the SQLite
# driver is modernc, pure Go, so nothing needs cgo).
ENV CGO_ENABLED=0
RUN bash scripts/build.sh

# The hosted CashFlux frontend, staged into web/cashflux/ the same way update.sh does it.
RUN CASHFLUX_SRC=/src/CashFlux bash scripts/build-cashflux.sh

# Assemble the exact release layout deploy/update.sh stages — binary beside web/, because
# the server resolves web/static and web/cashflux relative to its working directory.
RUN mkdir -p /release/web \
    && cp bin/server /release/server \
    && cp -a web/static /release/web/static \
    && cp -a web/cashflux /release/web/cashflux

FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=build /release/ /app/

# /data is where DB_PATH and CASHFLUX_DATA_DIR point in the container (compose bind-mounts
# the droplet's existing /opt/earlcameron/data here, so a cutover carries every row over
# and a rollback to the systemd unit sees the same files).
VOLUME ["/data"]

# Inside the container the server must bind all interfaces — compose publishes the port to
# the HOST's loopback only, which is where the "never expose :8095 past nginx" rule is
# enforced now. A container binding 127.0.0.1 would be unreachable from the host entirely.
ENV LISTEN_ADDR=:8095 \
    DB_PATH=/data/site.db \
    CASHFLUX_DATA_DIR=/data/cashflux
EXPOSE 8095

USER 65532:65532

# The binary checks itself (`server healthcheck` → GET /healthz): distroless has no shell
# or curl, and /healthz is the same endpoint the legacy deploy's rollback logic gated on.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/server", "healthcheck"]

ENTRYPOINT ["/app/server"]
