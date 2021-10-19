# syntax=docker/dockerfile:1
FROM golang:bullseye AS build
COPY . /go/src/cb/
RUN cd /go/src/cb && go build -o /bin/cb

# This results in a single layer image
FROM scratch
COPY --from=build /bin/cb /bin/cb
ENTRYPOINT ["/bin/cb"]
