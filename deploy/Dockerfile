FROM golang:1.22-alpine AS backend

RUN apk --no-cache add bash git openssh

WORKDIR /app

# cache dependencies
COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

# build
COPY . .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o killgravelogger -ldflags "-w -s" .


FROM alpine:3.16 AS prod

WORKDIR /app

COPY --from=backend /app/killgravelogger /app/

CMD [ "/app/killgravelogger" ]
