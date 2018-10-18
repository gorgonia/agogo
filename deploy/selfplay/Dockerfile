FROM ubuntu:16.04

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
        curl \
        git \
        pkg-config \
        rsync \
        awscli \
        wget \
        && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN wget -nv https://storage.googleapis.com/golang/go1.10.1.linux-amd64.tar.gz && \
        tar -C /usr/local -xzf go1.10.1.linux-amd64.tar.gz

ENV GOPATH /home/ubuntu/go

ENV GOROOT /usr/local/go

ENV PATH $PATH:$GOROOT/bin

RUN git clone https://user:pass@github.com/chewxy/agogo.git

RUN /usr/local/go/bin/go version && \
        echo $GOPATH && \
        echo $GOROOT

RUN go get -v gorgonia.org/gorgonia && \
      go get -v gorgonia.org/tensor && \
      go get -v gorgonia.org/dawson && \
      go get -v github.com/gogo/protobuf/gogoproto && \
      go get -v github.com/golang/protobuf/proto && \
      go get -v github.com/google/flatbuffers/go

WORKDIR /

ADD staging/ /app

WORKDIR /app

CMD ["/bin/sh", "player_wrapper.sh"]
