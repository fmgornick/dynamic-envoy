# dynamic proxy configuration generator

This program only works for envoy proxy now, but the goal is to make it extensible for any type of proxy.


## overview

Assuming we're using envoy proxy, you can run envoy to listen for incoming traffic and route to specific upstream clusters.  Users can provide configuration (for now, only in the form of a databag), and this application can send it to envoy at runtime, so envoy doesn't need to be restarted.

You can see examples of how this application takes databag input in the form of json files [here](https://git.target.com/FletcherGornick/dynamic-proxy/tree/main/databags).

## requirements

1. Go 1.18+
2. envoy 1.22.2+
3. openssl 3.0.5+


## quick start

If you want to see a working example of this application, you can use our provided [example server](https://git.target.com/FletcherGornick/dynamic-proxy/blob/main/server/server.go) and [databags](https://git.target.com/FletcherGornick/dynamic-proxy/tree/main/databags).  To run it, you should first generate a certificate on the address you'd like to listen on for routing to HTTPS websites.  You can see how to do that [here](#ssl).  Once that's set, you can run this program like so...

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
go run server.go&
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


## <a name="ssl"></a> generating SSL certificate for HTTPS connection

If you want to be able to route to secure addresses using https, then you need to run the following commands to create a certificate + key.
When running the line that creates the certificate signing request, you can just hit enter through everything, but put "hostname" for the "Common Name" option.
Replace "hostname" with whatever address you want to listen on.
```sh
openssl genrsa -out hostname.key 2048
openssl req -new -key hostname.key -out hostname.csr
openssl x509 -req -days 9999 -in hostname.csr -signkey hostname.key -out hostname.crt
```

You should also probably put these created files into a different directory
```sh
mv hostname.key hostname.crt hostname.csr /etc/ssl/certs/hostname.crt
```

If you're using curl to test the proxy, then you're also going to want to make sure curl knows where to look for your new certificate...
```sh
export CURL_CA_BUNDLE=/etc/ssl/certs/hostname.crt
```

If you're using a web browser to test it out then you're going to want to make sure your computer recognizes the certificate.  If you're using a mac, you can do this by going into the 'Keychain Access' app.  Navigate to 'System' on the left sidebar, then go to File -> Import Items...  It will then prompt you to add your hostname.crt file, so just choose it from where you created / moved it (if you put it in the etc folder, then you'll need to do 'CMD + SHIFT + .' to access files in /etc).  Once added, you need to select it and make sure to "Always Trust" the certificate.


## warning
If you're having the listener route to both HTTP and HTTPS depending on the path, then chrome might still tell you the address envoy is listening on is not secure, even if you have a certificate.  Chrome treats websites with mixed HTTP and HTTPS content as not secure.

## usage

The main application for this program is for websites with many upstream routes that are continuously changing and need constant proxy re-configuration.  With this program, you never need to stop the proxy.


## extension

If you would like to add to this project via adding configuration for other proxies, or accepting new user configurations, I tried my best to make this somewhat easily extensible.

For adding a new type of configuration, you just need to add a file in the [parser directory](https://git.target.com/FletcherGornick/dynamic-proxy/tree/main/utils/parser).  You just need to add implementation for turning the new config into a universal config that all proxies should be able to use defined [here](https://git.target.com/FletcherGornick/dynamic-proxy/blob/main/utils/config/universal/config.go).  You can see how I made the parser for databags [here](https://git.target.com/FletcherGornick/dynamic-proxy/blob/main/utils/parser/databag.go).

For adding a new proxy, you would need to add the new proxy config file (maybe some useful helper functions as well) in the [config/proxy directory](https://git.target.com/FletcherGornick/dynamic-proxy/tree/main/utils/config/proxy).  Then you'll also want to add a file to the [processor directory](https://git.target.com/FletcherGornick/dynamic-proxy/tree/main/utils/processor) to turn the universal configuration into a specific proxy configuration

