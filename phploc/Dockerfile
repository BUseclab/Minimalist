FROM ubuntu:latest

RUN apt-get update
RUN apt-get install -y phploc

WORKDIR /home/
ENTRYPOINT ["phploc", "/home/4.0.0"]
