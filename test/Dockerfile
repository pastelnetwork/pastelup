# we only need golang image to build the exectuable, then we create a slimmed ubuntu image and
# copy the pastelup exectuable into it

### BUILDER IMAGE ###
FROM golang:1.18 as builder
ARG GOLANG_WORK_DIR=/go/src/github.com/pasetlnetwork/pastelup/
WORKDIR ${GOLANG_WORK_DIR}

COPY . .
RUN go mod download
RUN go build .

### RUNTIME IMAGE ###
FROM ubuntu:20.04
ARG GOLANG_WORK_DIR=/go/src/github.com/pasetlnetwork/pastelup/

# TODO: remove vim (used for testing)
RUN apt-get update && apt-get install -y git wget curl sudo gnupg2 libgomp1 jq vim
RUN DEBIAN_FRONTEND=noninteractive TZ=Etc/UTC apt-get -y install tzdata

ENV USR_PW="ubuntu"
RUN useradd -rm -d /home/ubuntu -s /bin/bash -g root -G sudo -u 1001 ubuntu -p "$(openssl passwd -1 ubuntu)"

USER ubuntu
WORKDIR /home/ubuntu

# copy the executable into the bin folder so we can execute from anywhere
COPY  --from=builder ${GOLANG_WORK_DIR}pastelup /bin
