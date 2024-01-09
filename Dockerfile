FROM golang:1.21

WORKDIR /usr/src/app

# change go proxy
RUN go env -w GOPROXY=https://goproxy.cn

# change timezone
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o start ./...

CMD ["./start"]