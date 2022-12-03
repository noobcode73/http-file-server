package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Config struct {
	AllowCreatesFlag   bool
	AllowDeletesFlag   bool
	AllowUploadsFlag   bool
	CustomTemplateFlag string
	NoAllowHiddenFlag  bool
	PasswdFlag         string
	RootRoute          string
	SslCertificate     string
	SslKey             string
	Routes             Routes
	UserFlag           string
}

func NewConfig() Config {
	return Config{
		AllowCreatesFlag:   false,
		AllowDeletesFlag:   false,
		AllowUploadsFlag:   false,
		CustomTemplateFlag: "",
		NoAllowHiddenFlag:  false,
		RootRoute:          "/",
		SslCertificate:     "",
		SslKey:             "",
		UserFlag:           "",
		PasswdFlag:         "",
	}
}

func Run(addr string, cfg Config) error {
	mux := http.DefaultServeMux
	handlers := make(map[string]http.Handler)

	if len(cfg.Routes.Values) == 0 {
		_ = cfg.Routes.Set(".")
	}

	for _, route := range cfg.Routes.Values {
		handlers[route.Route] = &FileHandler{
			route:          route.Route,
			path:           route.Path,
			allowUpload:    cfg.AllowUploadsFlag,
			allowDelete:    cfg.AllowDeletesFlag,
			allowCreate:    cfg.AllowCreatesFlag,
			customTemplate: cfg.CustomTemplateFlag,
			noAllowHidden:  cfg.NoAllowHiddenFlag,
		}

		if cfg.UserFlag == "" && cfg.PasswdFlag == "" && route.User == "" && route.Passwd == "" {
			mux.Handle(route.Route, handlers[route.Route])
			log.Printf("serving local path %q on %q", route.Path, route.Route)
		} else {
			_user, _passwd := cfg.UserFlag, cfg.PasswdFlag
			if route.User != "" && route.Passwd != "" {
				_user, _passwd = route.User, route.Passwd
			}
			mux.HandleFunc(route.Route, BasicAuth(handlers[route.Route].ServeHTTP, _user, _passwd, cfg.CustomTemplateFlag))
			log.Printf("auth with serving local path %q on %q", route.Path, route.Route)
		}
	}

	_, rootRouteTaken := handlers[cfg.RootRoute]
	if !rootRouteTaken {
		route := cfg.Routes.Values[0].Route
		mux.Handle(cfg.RootRoute, http.RedirectHandler(route, http.StatusTemporaryRedirect))
		log.Printf("redirecting to %q from %q", route, cfg.RootRoute)
	}

	binaryPath, _ := os.Executable()
	if binaryPath == "" {
		binaryPath = "server"
	}
	if cfg.SslCertificate != "" && cfg.SslKey != "" {
		log.Printf("%s (HTTPS) listening on %q", filepath.Base(binaryPath), addr)
		return http.ListenAndServeTLS(addr, cfg.SslCertificate, cfg.SslKey, mux)
	}
	log.Printf("%s listening on %q", filepath.Base(binaryPath), addr)
	return http.ListenAndServe(addr, mux)
}
