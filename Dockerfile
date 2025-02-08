FROM golang AS build

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN go build -v -o app

FROM photon

WORKDIR /data

COPY ./web/static web/static
COPY ./web/templates web/templates
COPY --from=build /build/app /usr/local/bin

ENTRYPOINT ["app"]
