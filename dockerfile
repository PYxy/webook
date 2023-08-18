FROM golang:1.19.4 AS baseimage
WORKDIR $GOPATH/src/go_code
ENV GO111MODULE=on \
	GOOS=linux \
	GOARCH=amd64 \
        	CGO_ENABLED=0 \
        	GOPROXY=https://goproxy.io,direct

WORKDIR /src/cloudprimordial
COPY go.mod  go.sum ./
RUN go mod download && go mod verify && go mod tidy
COPY . .
RUN go build -o service ./cmd/*



FROM scratch
COPY --from=baseimage /src/cloudprimordial/service  /

ENTRYPOINT ["./service"]
