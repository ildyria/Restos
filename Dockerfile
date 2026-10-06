# syntax=docker/dockerfile:1

FROM node:24-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npx nuxt generate

FROM golang:1.26-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/frontend/.output/public/ ./internal/web/dist/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/restos ./cmd/restos \
 && mkdir /out/data

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=backend /out/restos /restos
COPY --from=backend --chown=nonroot:nonroot /out/data /data
ENV RESTOS_DATA_DIR=/data RESTOS_ADDR=:8080
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/restos"]
CMD ["serve"]
