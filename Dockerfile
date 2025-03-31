FROM golang:1.23 as build-env

ENV GO111MODULE=on        \
    CGO_ENABLED=0         \
    GOOS=linux            \
    GOARCH=amd64

WORKDIR /build
COPY . .

ARG VERSION
ARG APP_NAME
RUN go build -o app -ldflags "-X main.AppVersion=${VERSION}" cmd/${APP_NAME}/main.go

FROM alpine
WORKDIR /app
COPY --from=build-env /build/app .
COPY cmd/${APP_NAME}/application.yaml .
RUN chmod +x app

ENTRYPOINT ["/app/app"] 
