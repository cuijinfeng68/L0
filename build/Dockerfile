FROM golang:latest
RUN go get -u github.com/kardianos/govendor  \
    && mkdir -p $GOPATH/src/github.com/bocheninc/ \
    && cd $GOPATH/src/github.com/bocheninc/ \
    && wget https://github.com/bocheninc/L0/archive/master.zip   \
    && mv L0-master L0 \
    && go install -tags=embed ./cmd/lcnd/...
CMD ["lcnd"]
