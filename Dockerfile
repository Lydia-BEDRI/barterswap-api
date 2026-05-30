FROM golang:1.22-alpine

ARG UID=1000
ARG GID=1000

RUN addgroup -g ${GID} app && adduser -D -u ${UID} -G app app

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

USER app

EXPOSE 8080

CMD ["go", "run", "."]
