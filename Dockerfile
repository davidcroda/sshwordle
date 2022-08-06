# Builder image
FROM golang

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /tmp/sshwordle

FROM scratch 

WORKDIR /app

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /tmp/sshwordle /app/sshwordle
COPY --from=0 /app/migrations/ /app/migrations/

CMD ["/app/sshwordle", "--host", "0.0.0.0"]
