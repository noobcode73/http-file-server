# http-file-server

`http-file-server` is a dependency-free HTTP file server. Beyond directory listings and file downloads, it lets you download a whole directory as as `.zip` or `.tar.gz` (generated on-the-fly).

![screenshot](doc/screenshot.png)

## Contents

- [Contents](#contents)
- [Examples](#examples)
  - [Serving a path at `/`](#serving-a-path-at-)
  - [Serving $PWD at `/`](#serving-pwd-at-)
  - [Serving multiple paths, setting the HTTP port via CLI arguments](#serving-multiple-paths-setting-the-http-port-via-cli-arguments)
  - [Setting the HTTP port via environment variables](#setting-the-http-port-via-environment-variables)
  - [Uploading files using cURL](#uploading-files-using-curl)
  - [HTTPS (SSL/TLS)](#https-ssltls)
  - [Custom Templates](#templates)
  - [Create new folder](#new-folder)
  - [Disable show hidden files or dirs](#hidden)
  - [Auth](#auth)
  - [Auth single route](#auth-route)
- [Get it](#get-it)
  - [Using `go get`](#using-go-get)
  - [Pre-built binary](#pre-built-binary)
- [Use it](#use-it)

## Examples

### Serving a path at `/`

```sh
$ http-file-server /tmp
2018/11/13 23:00:03 serving local path "/tmp" on "/tmp/"
2018/11/13 23:00:03 redirecting to "/tmp/" from "/"
2018/11/13 23:00:03 http-file-server listening on ":8080"
```

### Serving $PWD at `/`

```sh
$ cd /tmp
$ http-file-server
2018/12/13 03:18:00 serving local path "/tmp" on "/tmp/"
2018/12/13 03:18:00 redirecting to "/tmp/" from "/"
2018/12/13 03:18:00 http-file-server listening on ":8080"
```

### Serving multiple paths, setting the HTTP port via CLI arguments

```sh
$ http-file-server -p 1234 /1=/tmp /2=/var/tmp
2018/11/13 23:01:44 serving local path "/tmp" on "/1/"
2018/11/13 23:01:44 serving local path "/var/tmp" on "/2/"
2018/11/13 23:01:44 redirecting to "/1/" from "/"
2018/11/13 23:01:44 http-file-server listening on ":1234"
```

### Setting the HTTP port via environment variables

```sh
$ export PORT=9999
$ http-file-server /abc/def/ghi=/tmp
2018/11/13 23:05:52 serving local path "/tmp" on "/abc/def/ghi/"
2018/11/13 23:05:52 redirecting to "/abc/def/ghi/" from "/"
2018/11/13 23:05:52 http-file-server listening on ":9999"
```

### Uploading files using cURL

```sh
$ http-file-server -uploads /=/path/to/serve
2020/03/10 22:00:54 serving local path "/path/to/serve" on "/"
2020/03/10 22:00:54 http-file-server listening on ":8080"
```

```sh
curl -LF "file=@example.txt" localhost:8080/path/to/upload/to
```

### HTTPS (SSL/TLS)

To terminate SSL at the file server, set `-ssl-cert` (`SSL_CERTIFICATE`) and `-ssl-key` (`SSL_KEY`) to the respective files' paths:

```sh
$ ./http-file-server -port 8443 -ssl-cert server.crt -ssl-key server.key
2020/03/10 22:00:54 http-file-server (HTTPS) listening on ":8443"
```

### Custom templates

![screenshot](doc/custom%20template.jpg)

Create a folder and add base.html

Create a subfolder 'errors' and add html files named 'status code': 401.html, 404.html, 500.html...

example templates in /templates/

```sh
$ ./http-file-server -t ./templates
2022/12/02 00:27:24 Added custom templates: ./templates
```

### Create new folder
Set `-c` or `--creates`
The new directory will be created in the current directory with permissions `665`

Note: html method `PUT` is used

```sh
$ ./http-file-server -c ./
```

### Disable show hidden files or dirs
You can disable the display of hidden files or directories using the `-nh` or `--nohidden` argument

Note. Does not affect downloading a directory as an archive.
```sh
$ ./http-file-server -nh ./                                                                   
```

### Auth
You can set Basic authorization for all routes using `--user` and `--passwd`
```sh
$ ./http-file-server --user admin --passwd 123456 ./                                         
```

### Auth single route
You can set Basic authorization for a single route using the format `user:passwd@/locatin=./local_path`
```sh
$ ./http-file-server admin:1234@/home=/tmp/home
```

Note: The `--user` and `--passwd` arguments will be ignored.

For example:
```sh
$ ./http-file-server --user user1 --passwd 112233 admin:1234@/main=/tmp/home /home=/test2 /shara=/srv/shara
```
Here, for all routes, except for `/main`, global authorization will apply (`--user` (`user1`) `--passwd` (`112233`))




## Get it

### Using `go install`

```sh
go install github.com/noobcode73/http-file-server@latest
```

After this the executable is installed in go's normal directorys (see ```go help install``` for more information)

## Use it

```text
GOPATH/http-file-server [OPTIONS] [[ROUTE=]PATH] [[ROUTE=]PATH...]
```

```text
Usage of http-file-server:
   -a string
        (alias for -addr) (default ":8080")
  -addr string
        address to listen on (environment variable "ADDR") (default ":8080")
  -c    (alias for -creates)
  -creates
        allow creates folder (environment variable "CREATES")
  -d    (alias for -deletes)
  -deletes
        allow deletes (environment variable "DELETES")
  -nh
        (alias for -nohidden)
  -nohidden
    no allow hidden folders or files (environment variable "NO_HIDDEN")
  -p int
     (alias for -port)
  -passwd string
    global password for all routes (without auth) (environment variable "PASSWD").
  -port int
     port to listen on (overrides -addr port) (environment variable "PORT")
  -q (alias for -quiet)
  -quiet
     disable all log output (environment variable "QUIET")
  -r value
     (alias for -route)
  -route value
      a route definition ROUTE=PATH (ROUTE defaults to basename of PATH if omitted)
  -ssl-cert string
      path to SSL server certificate (environment variable "SSL_CERTIFICATE")
  -ssl-key string
     path to SSL private key (environment variable "SSL_KEY")
  -t string
     (alias for -template)
  -templates string
        path to custom Templates folder html.
                base template = base.html, errors template = "status_code".html (401.html, 404.html, etc.).
                (environment variable "TEMPLATES")
  -u    (alias for -uploads)
  -uploads
        allow uploads (environment variable "UPLOADS")
  -user string
        global user name for all routes (without auth) (environment variable "USER").
```
