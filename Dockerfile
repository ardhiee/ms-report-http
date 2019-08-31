# build stage
FROM golang:alpine AS build-env
RUN apk --no-cache add build-base git bzr mercurial gcc

ADD . /go/src/ms-report-http
WORKDIR /go/src/ms-report-http
RUN go get ms-report-http
RUN go install
RUN cd /go/src/ms-report-http && go build -o ms-report-http

# final stage
FROM alpine
WORKDIR /go/src/ms-report-http
COPY --from=build-env /go/src/ms-report-http /go/src/ms-report-http
ENTRYPOINT ./ms-report-http
EXPOSE 8888
