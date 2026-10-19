FROM node:20 as buildweb

WORKDIR /app/web
COPY ./web/package.json /app/web/package.json
RUN npm install --registry=https://registry.npmmirror.com
COPY ./web .
COPY ./web/.env.prod ./.env
RUN npm run build

FROM golang:1.23-alpine as buildapi

WORKDIR /app/api
COPY ./api .
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download
RUN mkdir -p ./dist && CGO_ENABLED=0 go build -o ./dist/mishosapi .

FROM caddy:alpine

ARG S6_OVERLAY_VERSION=3.2.0.0

COPY ./docker/caddy/Caddyfile /etc/caddy/Caddyfile
COPY --from=buildweb /app/web/dist /app/web
COPY --from=buildapi /app/api/dist /app/api

ADD https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-noarch.tar.xz /tmp
RUN tar -C / -Jxpf /tmp/s6-overlay-noarch.tar.xz
ADD https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-x86_64.tar.xz /tmp
RUN tar -C / -Jxpf /tmp/s6-overlay-x86_64.tar.xz

COPY ./docker/s6/web/run /etc/services.d/web/run
COPY ./docker/s6/api/run /etc/services.d/api/run
RUN chmod +x /etc/services.d/web/run /etc/services.d/api/run

ENTRYPOINT ["/init"]
