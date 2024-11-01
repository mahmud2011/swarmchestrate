FROM golang:1.24.2-alpine AS builder

ENV GOOS=linux
ENV CGO_ENABLED=0

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -a -installsuffix 'static' -o /build/resource-lead-agent .

FROM scratch
COPY --from=builder /build/resource-lead-agent /resource-lead-agent

ENTRYPOINT ["/resource-lead-agent"]
