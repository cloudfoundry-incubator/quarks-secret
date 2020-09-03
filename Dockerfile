ARG BASE_IMAGE=registry.opensuse.org/cloud/platform/quarks/sle_15_sp1/quarks-operator-base:latest

FROM golang:1.14.7 AS build
ARG GOPROXY
ENV GOPROXY $GOPROXY

WORKDIR /go/src/code.cloudfoundry.org/quarks-secret

# Copy the rest of the source code and build
COPY . .
RUN [ -f tools/quarks-utils/bin/include/versioning ] || \
    bin/tools
RUN bin/build && \
    cp -p binaries/quarks-secret /usr/local/bin/quarks-secret

FROM $BASE_IMAGE
RUN groupadd quarks && \
    useradd -r -g quarks quarks
USER quarks
COPY --from=build /usr/local/bin/quarks-secret /usr/local/bin/quarks-secret
ENTRYPOINT ["/tini", "--", "/usr/local/bin/quarks-secret"]
