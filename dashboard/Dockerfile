FROM node:20.19.5-alpine3.22 AS builder

WORKDIR /app

COPY dashboard/package.json dashboard/package-lock.json ./
RUN npm ci

COPY dashboard/ ./
RUN npm run build

FROM nginx:1.29.1-alpine3.22

COPY dashboard/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1/healthz || exit 1
