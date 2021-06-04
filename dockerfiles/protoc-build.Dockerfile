FROM centos:8 AS protoc-build
LABEL maintainer="sayuri556677@gmail.com"

ENV GOPROXY https://goproxy.cn
ENV GOPATH /root/go
ENV GOROOT /usr/local/go
ENV PATH $PATH:$GOROOT/bin:$GOPATH/bin:/usr/local/node/bin
ENV PROTOBUF_VERSION 3.17.2
ENV GOLANG_VERSION 1.16.5
ENV NODE_VERSION 16.3.0

RUN yum -y update && yum -y install wget gcc automake autoconf libtool make gcc-c++ git zip && yum clean all
# protobuf
RUN wget -O protobuf.tar.gz https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOBUF_VERSION}/protobuf-all-${PROTOBUF_VERSION}.tar.gz
RUN tar -zxf protobuf.tar.gz && cd protobuf-${PROTOBUF_VERSION} && ./configure && make && make install && cd / && rm -rf protobuf.tar.gz protobuf-${PROTOBUF_VERSION}
# golang
RUN wget -O wget -O golang.tar.gz https://studygolang.com/dl/golang/go${GOLANG_VERSION}.linux-amd64.tar.gz
RUN tar -C /usr/local -zxf golang.tar.gz && rm -f golang.tar.gz
RUN go get github.com/golang/protobuf/{proto,protoc-gen-go} github.com/favadi/protoc-go-inject-tag
# node
RUN wget -O node.tar.gz https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.gz
RUN tar -C /usr/local -zxf node.tar.gz && rm -f node.tar.gz && mv /usr/local/node-v${NODE_VERSION}-linux-x64 /usr/local/node
RUN npm install -g protobufjs protoc-gen-ts grpc-tools grpc_tools_node_protoc_ts

CMD ["protoc"]
