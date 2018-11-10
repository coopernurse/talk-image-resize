FROM debian:9.5

RUN apt-get update && apt-get install -y imagemagick graphicsmagick time git \
   build-essential autoconf libtool gtk-doc-tools curl swig gobject-introspection \
   libjpeg-dev libjpeg62-turbo-dev librsvg2-dev libpoppler-dev libtiff5-dev liblcms2-dev \
   liborc-0.4-dev libpng-dev libwebp-dev libgif-dev libexpat1-dev libexif-dev libpoppler-glib-dev
   
RUN cd /tmp && curl -LO https://github.com/jcupitt/libvips/archive/v8.7.0.tar.gz && \
    tar zxf v8.7.0.tar.gz && \
    cd libvips-8.7.0 && \
    ./autogen.sh && make && make install && \
    cd /tmp && rm -rf libvips-8.7.0 && rm -f v8.7.0.tar.gz
ENV LD_LIBRARY_PATH=/usr/local/lib

RUN cd /usr/local && curl -LO https://dl.google.com/go/go1.11.2.linux-amd64.tar.gz && \
    tar zxf go1.11.2.linux-amd64.tar.gz && \
    rm -f go1.11.2.linux-amd64.tar.gz
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin
ENV GOROOT=/usr/local/go

RUN go get -u github.com/labstack/echo/... 
ADD . /app
RUN cd /app && mkdir -p /app/output && \
   go build -o bin/ui src/ui.go && \
   go build -o bin/resize src/resize.go

WORKDIR /app
