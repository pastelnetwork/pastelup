FROM ubuntu:20.04
RUN  apt-get update \
  && apt-get install -y wget \
  && apt-get -qq -y install curl\
  && apt-get install -y libgomp1\
  && apt-get install -y iputils-ping\
  && apt-get install -y zip unzip php-zip\
  && apt-get install -y ssh\
  && apt-get install -y expect\
  && apt-get install -y sed\
  && apt-get install -y passwd\
  && apt-get install -y sudo\
  && apt-get install -y python3-pip\
  && apt-get install -y dpkg\
  && apt-get install -y xdg-utils\
  && apt-get install -y libxshmfence1\
  && apt-get install -y libxrandr2\
  && apt-get install -y libxkbcommon0\
  && apt-get install -y libxfixes3\
  && apt-get install -y libxdamage1\
  && apt-get install -y libxcomposite1\
  && apt-get install -y libpango-1.0-0\
  && apt-get install -y libnss3\
  && apt-get install -y libnspr4\
  && apt-get install -y libgtk-3-0\
  && apt-get install -y libgbm1\
  && apt-get install -y libcups2\
  && apt-get install -y libcairo2\
  && apt-get install -y libatspi2.0-0\
  && apt-get install -y libatk1.0-0\
  && apt-get install -y libatk-bridge2.0-0\
  && apt-get install -y libasound2\
  && apt-get install -y fonts-liberation\  
  && apt-get install -y ufw\
  && rm -rf /var/lib/apt/lists/*
COPY ./pastel-utility /
COPY ./ssh_config /etc/ssh/
COPY ./sshd_config /etc/ssh/
RUN chmod 777 /pastel-utility


EXPOSE 22
