FROM golang

# set up app in /home/user/app
RUN mkdir -p /home/user/app
WORKDIR /home/user/app

# put all relevant files into container
COPY app app
COPY go.mod go.sum main.go .

# build and run app
RUN go build
ENTRYPOINT ["./dynamic-proxy"]
