# syntax=docker/dockerfile:1

# Static site generation only produces files, so it always runs natively on the
# build host instead of being emulated through QEMU for the target platform.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS site

WORKDIR /src

# Module manifests first so the dependency download caches across source edits.
COPY tools/docgen/go.mod tools/docgen/go.sum ./tools/docgen/
COPY tools/sitemap/go.mod ./tools/sitemap/
RUN cd tools/docgen && go mod download \
 && cd ../sitemap && go mod download

COPY tools/docgen ./tools/docgen
COPY tools/sitemap ./tools/sitemap
COPY web ./web

# Generated output is always rebuilt here — a developer's local copies may
# already carry an injected version and must never reach the image.
RUN find ./web/docs -name '*.html' ! -name '_template.html' -delete \
 && rm -f ./web/sitemap.xml

RUN cd tools/docgen && go run . -docs /src/web/docs
RUN cd tools/sitemap && go run . -docs /src/web/docs -out /src/web/sitemap.xml

ARG VERSION=dev
# web/index.html is a tracked source file that `task web:dev` rewrites in place.
# If that rewrite was never reverted the placeholder is gone and the injection
# below would silently ship a stale version, so fail the build instead.
RUN grep -q '__VERSION__' ./web/index.html \
 && find ./web -name '*.html' -exec sed -i "s/__VERSION__/${VERSION}/g" {} +

# Drop the docs sources so only the served files remain.
RUN find ./web -name '*.md' -delete \
 && find ./web -name '.DS_Store' -delete \
 && rm -f ./web/docs/_template.html

# Smallest official nginx variant: no perl/geoip modules, no njs.
FROM nginx:1.30-alpine-slim

RUN rm -rf /usr/share/nginx/html/*

COPY .docker/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=site /src/web/ /usr/share/nginx/html/

EXPOSE 80

# 127.0.0.1 rather than localhost: the probe must not depend on how the
# container resolves that name.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1/ || exit 1
