# syntax=docker/dockerfile:1-experimental

FROM golang 

# Set the PATH to include the new Go install.
ENV PATH="${PATH}:/usr/local/go/bin"

# Install custom version of go with larger minimum stack size.
RUN git clone https://github.com/ArielSzekely/go.git go-custom
RUN cd go-custom && \
  git checkout bigstack && \
  git config pull.rebase false && \
  git pull && \
  cd src && \
  ./make.bash

RUN git config --global http.sslverify false
WORKDIR /go/src/github.com/harlow/go-micro-services
COPY go.* .
RUN go mod download
COPY . /go/src/github.com/harlow/go-micro-services
RUN /go/go-custom/bin/go mod vendor

RUN --mount=type=cache,target=/root/.cache/go-build /go/go-custom/bin/go install -ldflags="-s -w" ./cmd/...
