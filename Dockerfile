FROM golang:latest
ENV APP_DIR $GOPATH/src/vBlockchain
ADD . $APP_DIR
WORKDIR $GOPATH/src/vBlockchain
RUN go get github.com/boltdb/bolt && go get golang.org/x/crypto/ripemd160 && go build -o vchain
CMD vBlockchain startnode