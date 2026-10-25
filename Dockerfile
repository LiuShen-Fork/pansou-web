# syntax=docker/dockerfile:1

# ---------- 前端构建 ----------
FROM node:22-alpine AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# ---------- 后端构建 ----------
FROM golang:1.25-alpine AS backend-builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG BUILDTIME=unknown
ARG REVISION=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/pansou .

# ---------- 运行镜像 ----------
FROM nginx:alpine AS runtime

RUN apk add --no-cache ca-certificates tzdata curl && \
    apk add --no-cache bash --allow-untrusted 2>/dev/null || \
    (apk add --no-cache --force-broken-world bash || true)

ENV TZ=Asia/Shanghai \
    PANSOU_PORT=8888 \
    PANSOU_HOST=127.0.0.1 \
    CACHE_PATH=/app/data/cache \
    LOG_PATH=/app/data/logs \
    ADMIN_DB_PATH=/app/data/pansou-admin.db \
    CACHE_ENABLED=true \
    CACHE_TTL=60 \
    MAX_CONCURRENCY=200 \
    MAX_PAGES=30 \
    HEALTH_CHECK_INTERVAL=30 \
    HEALTH_CHECK_TIMEOUT=10 \
    HEALTH_CHECK_RETRIES=3

WORKDIR /app

COPY --from=backend-builder /out/pansou /app/pansou
COPY --from=frontend-builder /src/frontend/dist /app/frontend/dist
COPY start.sh /app/start.sh
COPY deploy/upstream-defaults.env /app/upstream-defaults.env
RUN chmod +x /app/pansou /app/start.sh

# 健康检查脚本 - 同时确认 nginx 与后端可用
RUN cat > /app/healthcheck.sh << 'EOF'
#!/bin/bash
PANSOU_HOST=${PANSOU_HOST:-127.0.0.1}
PANSOU_PORT=${PANSOU_PORT:-8888}
HEALTH_CHECK_TIMEOUT=${HEALTH_CHECK_TIMEOUT:-10}

if ! pgrep nginx >/dev/null 2>&1; then
    echo "Nginx 进程不存在"
    exit 1
fi
if ! curl -sf --max-time "${HEALTH_CHECK_TIMEOUT}" http://localhost/api/health >/dev/null 2>&1; then
    echo "通过 Nginx 访问健康检查失败"
    exit 1
fi
if ! curl -sf --max-time "${HEALTH_CHECK_TIMEOUT}" "http://${PANSOU_HOST}:${PANSOU_PORT}/api/health" >/dev/null 2>&1; then
    echo "后端健康检查失败"
    exit 1
fi
exit 0
EOF
RUN chmod +x /app/healthcheck.sh

RUN mkdir -p /app/data/cache \
             /app/data/logs/backend \
             /app/data/logs/nginx \
             /app/data/ssl

HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD /app/healthcheck.sh || exit 1

EXPOSE 80 443
VOLUME ["/app/data"]

LABEL org.opencontainers.image.title="PanSou Web" \
      org.opencontainers.image.description="PanSou 前后端一体镜像，内置账户、OAuth2 与管理后台" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILDTIME}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.source="https://github.com/willow-god/pansou-web"

CMD ["/app/start.sh"]
