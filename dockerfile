FROM golang

RUN mkdir -p /home/user/app
COPY . /home/user/app
WORKDIR /home/user/app

RUN go build
ENTRYPOINT ["/home/user/app/dynamic-proxy"]
