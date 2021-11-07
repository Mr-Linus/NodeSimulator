FROM debian:stretch-slim

WORKDIR /

ADD bin/manager .
CMD ["/manager"]