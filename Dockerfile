# Builder image
FROM golang

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /tmp/sshwordle

FROM scratch 

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /tmp/sshwordle /app/sshwordle

CMD ["/app/sshwordle", "--host", "0.0.0.0"]
