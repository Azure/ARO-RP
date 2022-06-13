FROM golang AS builder

WORKDIR app
ADD go.mod go.sum .
RUN go mod download

ADD *.go .
RUN CGO_ENABLED=0 go build  -o /app

FROM alpine 
RUN apk add ca-certificates

COPY --from=builder /app /app
ADD example.* /
RUN chmod 644 /example.*
CMD ["/app"]
