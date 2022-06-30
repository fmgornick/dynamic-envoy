# dynamic proxy configuration generator

This program only works for envoy proxy now, but the goal is to make it extensible for any type of proxy.


## overview

Assuming we're using envoy proxy, you can run envoy to listen for incoming traffic and route to specific upstream clusters.  Users can provide configuration (for now, only in the form of a databag), and this application can send it to envoy at runtime, so envoy doesn't need to be restarted.

You can see examples of how this application takes databag input in the form of json files [here](https://github.com/fmgornick/dynamic-envoy/tree/main/databags).

## requirements

1. Go 1.18+
2. envoy 1.22.2+


## quick start

If you want to see a working example of this application, you can use our provided [example server](https://github.com/fmgornick/dynamic-envoy/blob/main/server/server.go) and [databags](https://github.com/fmgornick/dynamic-envoy/tree/main/databags/local) like so...

1. clone this repository:
```sh
git clone git@github.com:fmgornick/dynamic-envoy
```

2. navigate to example server directory:
```sh
cd dynamic-envoy/server
```

3. run server in background (should run on localhost, ports 1111, 2222, 3333, and 4444):
```sh
go run server.go &
```

4. navigate to envoy bootstrap configuration directory:
```sh
cd ../bootstrap
```

5. start envoy server (can also run in background with \'&\' suffix):
```sh
envoy -c bootstrap.yaml
```

6. navigate back to root of project and run application (use \'-h\' to see possible flags):
```sh
go run main.go
# or
go build main.go
./main
```

7.  If no flags changed, go to http://localhost:7777 or http://localhost:8888.

8. you can move / delete / add files to the directory the application is watching and see as the envoy configuration updates real time.


## usage

The main application for this program is for websites with many upstream routes that are continuously changing and need constant proxy re-configuration.  With this program, you never need to stop the proxy.


## extension

If you would like to add to this project via adding configuration for other proxies, or accepting new user configurations, I tried my best to make this somewhat easily extensible.

For adding a new type of configuration, you just need to add a file in the [parser directory](https://github.com/fmgornick/dynamic-envoy/tree/main/utils/parser).  You just need to add implementation for turning the new config into a universal config that all proxies should be able to use defined [here](https://github.com/fmgornick/dynamic-envoy/blob/main/utils/config/universal/config.go).  You can see how I made the parser for databags [here](https://github.com/fmgornick/dynamic-envoy/blob/main/utils/parser/databag.go).

For adding a new proxy, you would need to add the new proxy config file (maybe some useful helper functions as well) in the [config/proxy directory](https://github.com/fmgornick/dynamic-envoy/tree/main/utils/config/proxy).  Then you'll also want to add a file to the [processor directory](https://github.com/fmgornick/dynamic-envoy/tree/main/utils/processor) to turn the universal configuration into a specific proxy configuration
