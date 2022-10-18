FROM golang:1.19 as base

FROM base as dev

WORKDIR /opt/app
CMD ["main"]