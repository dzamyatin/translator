FROM ubuntu:24.04

RUN apt-get clean
RUN apt-get autoclean
#RUN apt-key update
RUN apt-get upgrade
RUN apt-get update
RUN apt-get install -y gawk
RUN apt-get install -y git make

RUN git clone https://github.com/soimort/translate-shell && cd translate-shell && \
    make && \
    make install

RUN apt-get install -y golang

COPY ./ /app

WORKDIR /app

RUN go build ./

CMD ./translator