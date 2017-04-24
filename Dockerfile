FROM golang:1.8.1-onbuild

RUN apt-get update && apt-get install -y webp optipng

RUN curl -O https://mozjpeg.codelove.de/bin/mozjpeg_3.1_amd64.deb && dpkg -i mozjpeg_3.1_amd64.deb

ENV PATH /opt/mozjpeg/bin:$PATH
