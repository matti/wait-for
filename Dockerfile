FROM golang:1.15.0-alpine3.12 as builder

COPY . /app/
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wait-for .

FROM scratch
COPY --from=builder /app/wait-for /wait-for
ENTRYPOINT [ "/wait-for" ]
