FROM golang:1.26-bookworm AS builder

LABEL maintainer="libreFS contributors"

ENV CGO_ENABLED=0

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath --ldflags "-s -w" -o /lc .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /lc /usr/bin/lc
COPY --from=builder /app/LICENSE /licenses/LICENSE
COPY --from=builder /app/CREDITS /licenses/CREDITS

ENTRYPOINT ["/usr/bin/lc"]
