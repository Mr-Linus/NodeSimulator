FROM debian:stretch-slim

WORKDIR /

COPY bin/manager /usr/local/bin

CMD ["manager"]