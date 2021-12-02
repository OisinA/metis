FROM golang:1.17.1

WORKDIR /metis

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build

FROM alpine
RUN apk update && apk add ca-certificates
COPY --from=0 /metis/metis .
ENTRYPOINT [ "./metis", "agent" ]