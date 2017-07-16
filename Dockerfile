FROM golang
ADD . /go/src/github.com/tobyjsullivan/event-log-reader
RUN  go install github.com/tobyjsullivan/event-log-reader

EXPOSE 3000

CMD /go/bin/event-log-reader
