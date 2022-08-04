FROM golang

# set up app in /home/user/app
RUN mkdir -p /home/user/app
WORKDIR /home/user/app

# put all relevant files into container
COPY app app
COPY go.mod go.sum main.go .
COPY add-cert.sh .

# create ssl certificate for proxy
RUN ./add-cert.sh localhost

# build and run app
RUN go build
ENTRYPOINT ["./dynamic-proxy"]
