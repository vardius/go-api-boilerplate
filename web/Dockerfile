# syntax=docker/dockerfile:1

FROM node:26.12.1-alpine3.24 AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY . .
RUN npm run build

# Runs as the image's unprivileged nginx user (UID 101) and listens on 8080.
FROM nginxinc/nginx-unprivileged:1.31.7-alpine3.24
LABEL org.opencontainers.image.source="https://github.com/vardius/go-api-boilerplate" \
      org.opencontainers.image.licenses="MIT"
ENV AUTH_UPSTREAM=http://auth:8080 \
    USER_UPSTREAM=http://user:8080
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
