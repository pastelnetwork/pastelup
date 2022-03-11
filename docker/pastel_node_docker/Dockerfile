FROM ubuntu:20.04
RUN  apt-get update \
  && apt-get install -y wget \
  && apt-get -qq -y install curl\
  && apt-get install -y libgomp1\
  && rm -rf /var/lib/apt/lists/*
COPY ./pastelup /
COPY ./node.sh /
RUN chmod 777 /node.sh
RUN chmod 777 /pastelup
ENTRYPOINT ["/node.sh"]






