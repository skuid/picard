# To build:
# $ docker run --rm -v $(pwd):/go/src/github.com/skuid/warden -w /go/src/github.com/skuid/warden golang:1.7  go build -v -a -tags netgo -installsuffix netgo -ldflags '-w'
# $ docker build -t skuid/warden .
#
# To run:
# $ docker run skuid/warden

FROM alpine

LABEL maintainer=matt.hardwick@skuid.com

RUN apk -U add ca-certificates

COPY warden /bin/warden
RUN chmod 755 /bin/warden

ADD migrations /migrations/

ADD swagger.json /bin/
RUN chmod 444 /bin/swagger.json

EXPOSE 3000

ENTRYPOINT ["/bin/warden"]
CMD ["serve"]
