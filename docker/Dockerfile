FROM centos@sha256:8d487d68857f5bc9595793279b33d082b03713341ddec91054382641d14db861 AS build

USER root

RUN yum -y update

# dev tools
RUN yum group install -y "Development Tools"

# Install Go
RUN curl -SL https://dl.google.com/go/go1.13.4.linux-amd64.tar.gz | tar -xzC /usr/local && mkdir -p /go/bin

ENV GO111MODULE=on
ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV GOBIN=$GOPATH/bin
ENV PATH="/go/bin:/usr/local/go/bin:${PATH}"

# Install bastet
RUN go get github.com/gdey/bastet/cmd/bastet && mkdir -p /etc/templates

RUN curl -s https://packagecloud.io/install/repositories/github/git-lfs/script.rpm.sh | bash && yum install -y git-lfs vim

COPY docker/files/templates /etc/templates

RUN yum install -y centos-release-scl
RUN yum install -y devtoolset-7-gcc* && /go/bin/bastet -o /etc/profile.d/devtoolset-7.sh /etc/templates/scl_profile.tpl "package=devtoolset-7"
RUN yum install -y rh-nodejs10-npm.x86_64 && /go/bin/bastet -o /etc/profile.d/rh-nodejs10.sh /etc/templates/scl_profile.tpl "package=rh-nodejs10"

# install cmake
RUN cd /usr/local;    curl -SL https://github.com/Kitware/CMake/releases/download/v3.13.4/cmake-3.13.4-Linux-x86_64.tar.gz |  tar -xzC /usr/local && ln -s cmake-3.13.4-Linux-x86_64 cmake; echo 'PATH=$PATH:/usr/local/cmake/bin' > /etc/profile.d/cmake.sh

ENV PATH="${PATH}:/usr/local/cmake/bin"

RUN yum install -y cairo-devel librsvg2-devel

RUN yum install -y gdk-pixbuf2-devel \
	pango-devel \
	libxml2-devel \
	libcroco-devel \
	libpng-devel \
	libjpeg-turbo-devel \
	gtk3-devel

RUN yum install -y sqlite-devel.x86_64 \
	mesa-libOSMesa-devel.x86_64 \
	libcurl-devel.x86_64

# copy source code to container
COPY . $GOPATH/src/github.com/go-spatial/atlante/

# build binary
WORKDIR $GOPATH/src/github.com/go-spatial/atlante/cmd/atlante/
RUN source ~/.bash_profile && go install

# build final container without build dependencies
FROM centos@sha256:8d487d68857f5bc9595793279b33d082b03713341ddec91054382641d14db861

USER root

RUN yum -y update
RUN yum install -y mesa-libOSMesa.x86_64 \
	libpng \
	libjpeg-turbo \
	librsvg2

# copy binary from build image
COPY --from=build /go/bin/atlante /usr/bin/atlante

ENTRYPOINT ["/usr/bin/atlante"]
