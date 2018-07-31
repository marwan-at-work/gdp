package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/marwan-at-work/gdp"
	"github.com/marwan-at-work/gdp/download"
)

const pathList = "/{module:.+}/@v/list"
const pathVersionModule = "/{module:.+}/@v/{version}.mod"
const pathVersionInfo = "/{module:.+}/@v/{version}.info"
const pathLatest = "/{module:.+}/@latest"
const pathVersionZip = "/{module:.+}/@v/{version}.zip"

var token = flag.String("token", "", "github token against rate limiting")

func getRedirectURL(path string) string {
	return "http://localhost:3000/" + strings.TrimPrefix(path, "/")
}

func main() {
	flag.Parse()
	r := mux.NewRouter()
	dp := download.New(*token)
	r.HandleFunc(pathList, func(w http.ResponseWriter, r *http.Request) {
		module, err := getModule(r)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(400)
			return
		}

		vers, err := dp.List(r.Context(), module)
		if err != nil {
			sc := statusErr(err)
			if sc == 404 {
				http.Redirect(w, r, getRedirectURL(r.URL.Path), http.StatusMovedPermanently)
				return
			}
			fmt.Println(err)
			w.WriteHeader(sc)
			return
		}

		fmt.Fprint(w, strings.Join(vers, "\n"))
	})

	r.HandleFunc(pathVersionModule, func(w http.ResponseWriter, r *http.Request) {
		module, ver, err := modAndVersion(r)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(400)
			return
		}
		bts, err := dp.GoMod(r.Context(), module, ver)
		if err != nil {
			sc := statusErr(err)
			if sc == 404 {
				http.Redirect(w, r, getRedirectURL(r.URL.Path), http.StatusMovedPermanently)
				return
			}
			fmt.Println(err)
			w.WriteHeader(sc)
			return
		}

		w.Write(bts)
	})

	r.HandleFunc(pathVersionInfo, func(w http.ResponseWriter, r *http.Request) {
		module, ver, err := modAndVersion(r)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(400)
			return
		}
		info, err := dp.Info(r.Context(), module, ver)
		if err != nil {
			sc := statusErr(err)
			if sc == 404 {
				http.Redirect(w, r, getRedirectURL(r.URL.Path), http.StatusMovedPermanently)
				return
			}
			fmt.Println(err)
			w.WriteHeader(sc)
			return
		}

		json.NewEncoder(w).Encode(info)
	})

	r.HandleFunc(pathLatest, func(w http.ResponseWriter, r *http.Request) {
		module, err := getModule(r)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(400)
			return
		}

		info, err := dp.Latest(r.Context(), module)
		if err != nil {
			sc := statusErr(err)
			if sc == 404 {
				http.Redirect(w, r, getRedirectURL(r.URL.Path), http.StatusMovedPermanently)
				return
			}
			fmt.Println(err)
			w.WriteHeader(sc)
			return
		}

		json.NewEncoder(w).Encode(info)
	})

	r.HandleFunc(pathVersionZip, func(w http.ResponseWriter, r *http.Request) {
		module, ver, err := modAndVersion(r)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(400)
			return
		}
		rdr, err := dp.Zip(r.Context(), module, ver, "")
		if err != nil {
			sc := statusErr(err)
			if sc == 404 {
				http.Redirect(w, r, getRedirectURL(r.URL.Path), http.StatusMovedPermanently)
				return
			}
			fmt.Println(err)
			w.WriteHeader(sc)
			return
		}

		io.Copy(w, rdr)
	})

	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.Method, r.URL.String())
			h.ServeHTTP(w, r)
		})
	})

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("NOT FOUND", r.URL.String())

		w.WriteHeader(http.StatusNotFound)
	})

	http.ListenAndServe(":8090", r)
}

func getModule(r *http.Request) (string, error) {
	str := mux.Vars(r)["module"]
	if str == "" {
		return "", fmt.Errorf("missing module in path")
	}

	return DecodePath(str)
}

func getVersion(r *http.Request) (string, error) {
	str := mux.Vars(r)["version"]
	if str == "" {
		return "", fmt.Errorf("missing version in path")
	}

	return DecodeVersion(str)
}

func modAndVersion(r *http.Request) (mod, ver string, err error) {
	mod, err = getModule(r)
	if err != nil {
		return "", "", err
	}

	ver, err = getVersion(r)
	if err != nil {
		return "", "", err
	}

	return mod, ver, nil
}

func statusErr(err error) int {
	if err == gdp.ErrNotFound {
		return 404
	}

	return 500
}
