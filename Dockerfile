FROM golang:1.15-alpine AS build-stage

RUN apk add --no-cache git

WORKDIR /mdathome-golang

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o ./mdathome-golang -tags netgo -trimpath -ldflags '-s -w' ./cmd/mdathome


FROM alpine:3

RUN apk add --no-cache ca-certificates  

WORKDIR /mangahome

COPY --from=build-stage /mdathome-golang/mdathome-golang .
VOLUME /mangahome/cache

CMD ./mdathome-golang