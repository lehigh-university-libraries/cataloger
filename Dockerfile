FROM ghcr.io/lehigh-university-libraries/scyllaridae-imagemagick:main@sha256:d085b210070148e00a0605ba653e93bc6c6e6d8cfeea74b092a256203864f757

WORKDIR /app

RUN adduser -S -G nobody -u 8888 cataloger

COPY --chown=cataloger:cataloger main.go go.* docker-entrypoint.sh ./
COPY --chown=cataloger:cataloger internal/ ./internal/

RUN go mod download && \
  go build -o /app/cataloger && \
  go clean -cache -modcache

COPY --chown=cataloger:cataloger static/ ./static/

RUN mkdir uploads cache && \
  chown -R cataloger uploads cache

ENTRYPOINT ["/bin/bash"]
CMD ["/app/docker-entrypoint.sh"]

HEALTHCHECK CMD curl -fs http://localhost:8888/healthcheck
