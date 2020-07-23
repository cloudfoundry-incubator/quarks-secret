ARG BASE_IMAGE=registry.opensuse.org/cloud/platform/quarks/sle_15_sp1/quarks-operator-base:latest

FROM golang:1.13.4 AS build
ARG GOPROXY
ENV GOPROXY $GOPROXY
ARG GO111MODULE="on"
ENV GO111MODULE $GO111MODULE

WORKDIR /go/src/code.cloudfoundry.org/quarks-secret

# First, download dependencies so we can cache this layer
COPY go.mod .
COPY go.sum .
RUN if [ "${GO111MODULE}" = "on" ]; then go mod download; fi

# Copy the rest of the source code and build
COPY . .
RUN bin/build && \
    cp -p binaries/quarks-secret /usr/local/bin/quarks-secret

FROM $BASE_IMAGE
RUN groupadd quarks && \
    useradd -r -g quarks quarks
USER quarks
COPY --from=build /usr/local/bin/quarks-secret /usr/local/bin/quarks-secret
ENTRYPOINT ["/tini", "--", "/usr/local/bin/quarks-secret"]
