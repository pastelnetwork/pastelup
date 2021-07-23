FROM ubuntu:20.04
RUN  apt-get update \
  && apt-get install -y wget \
  && apt-get -qq -y install curl\
  && apt-get install -y libgomp1\
  && apt-get install -y iputils-ping\
  && apt-get install -y zip unzip php-zip\
  && apt-get install -y ssh\
  && apt-get install -y expect\
  && rm -rf /var/lib/apt/lists/*
COPY ./pastel-utility /
RUN chmod 777 /pastel-utility






