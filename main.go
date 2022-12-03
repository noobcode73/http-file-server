package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	addrEnvVarName           = "ADDR"
	allowUploadsEnvVarName   = "UPLOADS"
	allowDeletesEnvVarName   = "DELETES"
	allowCreatesEnvVarName   = "CREATES"
	noAllowHiddenEnvVarName  = "NO_HIDDEN"
	defaultAddr              = ":8080"
	portEnvVarName           = "PORT"
	quietEnvVarName          = "QUIET"
	rootRoute                = "/"
	customTemplateEnvVarName = "TEMPLATES"
	sslCertificateEnvVarName = "SSL_CERTIFICATE"
	sslKeyEnvVarName         = "SSL_KEY"
	userEnvVarName           = "USER"
	passwdEnvName            = "PASSWD"
)

var (
	addrFlag           = os.Getenv(addrEnvVarName)
	allowUploadsFlag   = os.Getenv(allowUploadsEnvVarName) == "true"
	allowDeletesFlag   = os.Getenv(allowDeletesEnvVarName) == "true"
	allowCreatesFlag   = os.Getenv(allowCreatesEnvVarName) == "true"
	noAllowHiddenFlag  = os.Getenv(noAllowHiddenEnvVarName) == "true"
	customTemplateFlag = os.Getenv(customTemplateEnvVarName)
	portFlag64, _      = strconv.ParseInt(os.Getenv(portEnvVarName), 10, 64)
	portFlag           = int(portFlag64)
	quietFlag          = os.Getenv(quietEnvVarName) == "true"
	routesFlag         routes
	sslCertificate     = os.Getenv(sslCertificateEnvVarName)
	sslKey             = os.Getenv(sslKeyEnvVarName)
	userFlag           = os.Getenv(userEnvVarName)
	passwdFlag         = os.Getenv(passwdEnvName)
)

func init() {
	log.SetFlags(log.LUTC | log.Ldate | log.Ltime)
	log.SetOutput(os.Stderr)
	if addrFlag == "" {
		addrFlag = defaultAddr
	}
	flag.StringVar(&addrFlag, "addr", addrFlag, fmt.Sprintf("address to listen on (environment variable %q)", addrEnvVarName))
	flag.StringVar(&addrFlag, "a", addrFlag, "(alias for -addr)")
	flag.IntVar(&portFlag, "port", portFlag, fmt.Sprintf("port to listen on (overrides -addr port) (environment variable %q)", portEnvVarName))
	flag.IntVar(&portFlag, "p", portFlag, "(alias for -port)")
	flag.BoolVar(&quietFlag, "quiet", quietFlag, fmt.Sprintf("disable all log output (environment variable %q)", quietEnvVarName))
	flag.BoolVar(&quietFlag, "q", quietFlag, "(alias for -quiet)")
	flag.BoolVar(&allowUploadsFlag, "uploads", allowUploadsFlag, fmt.Sprintf("allow uploads (environment variable %q)", allowUploadsEnvVarName))
	flag.BoolVar(&allowUploadsFlag, "u", allowUploadsFlag, "(alias for -uploads)")
	flag.BoolVar(&allowDeletesFlag, "deletes", allowDeletesFlag, fmt.Sprintf("allow deletes (environment variable %q)", allowDeletesEnvVarName))
	flag.BoolVar(&allowDeletesFlag, "d", allowDeletesFlag, "(alias for -deletes)")
	flag.BoolVar(&allowCreatesFlag, "creates", allowCreatesFlag, fmt.Sprintf("allow creates folder (environment variable %q)", allowCreatesEnvVarName))
	flag.BoolVar(&allowCreatesFlag, "c", allowCreatesFlag, "(alias for -creates)")
	flag.BoolVar(&noAllowHiddenFlag, "nohidden", allowCreatesFlag, fmt.Sprintf("no allow hidden folders or files (environment variable %q)", noAllowHiddenEnvVarName))
	flag.BoolVar(&noAllowHiddenFlag, "nh", allowCreatesFlag, "(alias for -nohidden)")
	flag.Var(&routesFlag, "route", routesFlag.help())
	flag.Var(&routesFlag, "r", "(alias for -route)")
	flag.StringVar(&sslCertificate, "ssl-cert", sslCertificate, fmt.Sprintf("path to SSL server certificate (environment variable %q)", sslCertificateEnvVarName))
	flag.StringVar(&sslKey, "ssl-key", sslKey, fmt.Sprintf("path to SSL private key (environment variable %q)", sslKeyEnvVarName))
	flag.StringVar(&customTemplateFlag, "templates", customTemplateFlag, fmt.Sprintf("path to custom Templates folder html.\n\tbase template = base.html, errors template = \"status_code\".html (401.html, 404.html, etc.).\n\t(environment variable %q)", customTemplateEnvVarName))
	flag.StringVar(&customTemplateFlag, "t", customTemplateFlag, "(alias for -template)")
	flag.StringVar(&userFlag, "user", userFlag, fmt.Sprintf("global user name for all routes (without auth) (environment variable %q).", userEnvVarName))
	flag.StringVar(&passwdFlag, "passwd", passwdFlag, fmt.Sprintf("global password for all routes (without auth) (environment variable %q).", passwdEnvName))
	flag.Parse()
	if quietFlag {
		log.SetOutput(ioutil.Discard)
	}
	for i := 0; i < flag.NArg(); i++ {
		arg := flag.Arg(i)
		err := routesFlag.Set(arg)
		if err != nil {
			log.Fatalf("%q: %v", arg, err)
		}
	}
}

func main() {
	addr, err := addr()
	if err != nil {
		log.Fatalf("address/port: %v", err)
	}
	err = server(addr, routesFlag)
	if err != nil {
		log.Fatalf("start server: %v", err)
	}
}

func server(addr string, routes routes) error {
	// check exist folder templates
	if customTemplateFlag != "" {
		if stat, err := os.Stat(customTemplateFlag); os.IsNotExist(err) || stat.IsDir() == false {
			log.Printf("Wrong path to folder with custom templates: %s\n", customTemplateFlag)
			customTemplateFlag = ""
		} else {
			if string(customTemplateFlag[len(customTemplateFlag)-1]) == osPathSeparator {
				customTemplateFlag = strings.TrimSuffix(customTemplateFlag, osPathSeparator)
			}
			log.Printf("Added custom templates: %s", customTemplateFlag)
		}
	}

	mux := http.DefaultServeMux
	handlers := make(map[string]http.Handler)

	if len(routes.Values) == 0 {
		_ = routes.Set(".")
	}

	for _, route := range routes.Values {
		handlers[route.Route] = &fileHandler{
			route:          route.Route,
			path:           route.Path,
			allowUpload:    allowUploadsFlag,
			allowDelete:    allowDeletesFlag,
			allowCreate:    allowCreatesFlag,
			customTemplate: customTemplateFlag,
			noAllowHidden:  noAllowHiddenFlag,
		}

		if userFlag == "" && passwdFlag == "" && route.User == "" && route.Passwd == "" {
			mux.Handle(route.Route, handlers[route.Route])
			log.Printf("serving local path %q on %q", route.Path, route.Route)
		} else {
			_user, _passwd := userFlag, passwdFlag
			if route.User != "" && route.Passwd != "" {
				_user, _passwd = route.User, route.Passwd
			}
			mux.HandleFunc(route.Route, BasicAuth(handlers[route.Route].ServeHTTP, _user, _passwd, "Please enter your username and password for this site"))
			log.Printf("auth with serving local path %q on %q", route.Path, route.Route)
		}
	}

	_, rootRouteTaken := handlers[rootRoute]
	if !rootRouteTaken {
		route := routes.Values[0].Route
		mux.Handle(rootRoute, http.RedirectHandler(route, http.StatusTemporaryRedirect))
		log.Printf("redirecting to %q from %q", route, rootRoute)
	}

	binaryPath, _ := os.Executable()
	if binaryPath == "" {
		binaryPath = "server"
	}
	if sslCertificate != "" && sslKey != "" {
		log.Printf("%s (HTTPS) listening on %q", filepath.Base(binaryPath), addr)
		return http.ListenAndServeTLS(addr, sslCertificate, sslKey, mux)
	}
	log.Printf("%s listening on %q", filepath.Base(binaryPath), addr)
	return http.ListenAndServe(addr, mux)
}

func addr() (string, error) {
	portSet := portFlag != 0
	addrSet := addrFlag != ""
	switch {
	case portSet && addrSet:
		a, err := net.ResolveTCPAddr("tcp", addrFlag)
		if err != nil {
			return "", err
		}
		a.Port = portFlag
		return a.String(), nil
	case !portSet && addrSet:
		a, err := net.ResolveTCPAddr("tcp", addrFlag)
		if err != nil {
			return "", err
		}
		return a.String(), nil
	case portSet && !addrSet:
		return fmt.Sprintf(":%d", portFlag), nil
	case !portSet && !addrSet:
		fallthrough
	default:
		return defaultAddr, nil
	}
}
