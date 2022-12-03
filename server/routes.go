package server

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Routes struct {
	Separator string

	Values []struct {
		Route  string
		Path   string
		User   string
		Passwd string
	}
	Texts []string
}

func (fv *Routes) Help() string {
	separator := "="
	if fv.Separator != "" {
		separator = fv.Separator
	}

	return fmt.Sprintf("a route definition ROUTE%sPATH (ROUTE defaults to basename of PATH if omitted)\nAdd a auth to /route: user:passwd@/route=/local_path", separator)
}

// getAuth parse flag.Value and return str without auth, userName, password
func getAuth(v string) (s, u, p string) {
	s = v
	if strings.Index(v, "@") > 0 && strings.Index(v, ":") > 0 {
		tmp := strings.Split(v, "@")
		u = tmp[0]
		// folder can start with '@'
		s = strings.Join(tmp[1:], "@")

		tmp = strings.Split(u, ":")
		u = tmp[0]
		// password may contain ':'
		p = strings.Join(tmp[1:], ":")

		if u != "" && p != "" {
			return s, u, p
		} else {
			fmt.Printf("user or password is empty: %s.\n\tuser: %s\n\tpassword: %s\n\texample: user:passwd@/location\n", v, u, p)
		}
	}
	// folder or url can`t start with ':'
	if strings.Index(s, ":") == 0 {
		s = strings.TrimPrefix(s, ":")
	}
	return s, "", ""
}

// Set is flag.Value.Set
func (fv *Routes) Set(v string) error {
	separator := "="
	if fv.Separator != "" {
		separator = fv.Separator
	}
	var route, path, user, password string
	var err error
	v, user, password = getAuth(v)

	i := strings.Index(v, separator)
	if i <= 0 {
		path = strings.TrimPrefix(v, "=")
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		route = fmt.Sprintf("/%s/", filepath.Base(path))
	} else {
		route = v[:i]
		path = v[i+len(separator):]
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(route, "/") {
			route = "/" + route
		}
		if !strings.HasSuffix(route, "/") {
			route = route + "/"
		}
	}
	fv.Texts = append(fv.Texts, v)
	fv.Values = append(fv.Values, struct {
		Route  string
		Path   string
		User   string
		Passwd string
	}{
		Route:  route,
		Path:   path,
		User:   user,
		Passwd: password,
	})
	return nil
}

func (fv *Routes) String() string {
	return strings.Join(fv.Texts, ", ")
}
