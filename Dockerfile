FROM golang:latest
#ENV NODE_ID=3000
COPY . $GOPATH/src/vBlockchain
WORKDIR $GOPATH/src/vBlockchain
RUN go get github.com/boltdb/bolt && go get golang.org/x/crypto/ripemd160
RUN go install ./...
#EXPOSE 3000
#CMD ["vBlockchain","startnode"]