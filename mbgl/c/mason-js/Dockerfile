# build with: `docker build -t mason-js .`
# run `docker run -it mason-js bash` 

FROM ubuntu:16.04

RUN apt-get update -y && apt-get install -y curl

# install node from node.js org
RUN curl https://nodejs.org/dist/v4.8.4/node-v4.8.4-linux-x64.tar.gz | tar zxC /usr/local --strip-components=1

RUN mkdir -p /tmp/mason-js-src
WORKDIR /tmp/mason-js-src
COPY . /tmp/mason-js-src

RUN npm install 
RUN npm link
CMD npm test