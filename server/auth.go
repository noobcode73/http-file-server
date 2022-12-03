package server

import (
	"crypto/subtle"
	"net/http"
	"os"
)

const realm = "Please enter your username and password for this site"

var template401 = []byte(`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Error 404</title>
  <style>
    .center-xy { top: 50%; left: 50%; transform: translate(-50%, -50%); position: absolute; }
    html, body { font-family: 'Roboto Mono', monospace; background-color: #000; box-sizing: border-box; user-select: none;}
    .container { width: 100%;text-align: center; }
    p { color: #fff; font-size: 24px; letter-spacing: .2px; margin: 0;}
  </style>
</head>
<body>
<div class="container">
  <div class="copy-container center-xy">
    <p>401, Unauthorized.</p>
    <br>
    <p><a href="/">Go home page</a></p>
  </div>
</div>
</body>
</html>`)

func BasicAuth(handler http.HandlerFunc, username, password, customTemplate string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.WriteHeader(401)
			if customTemplate != "" {
				p := customTemplate + osPathSeparator + "errors" + osPathSeparator + "401.html"
				template401, _ = os.ReadFile(p)
			}
			w.Write(template401)
			return
		}
		handler(w, r)
	}
}
