package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/marwan-at-work/gdp"
)

const pathList = "/{module:.+}/@v/list"
const pathVersionModule = "/{module:.+}/@v/{version}.mod"
const pathVersionInfo = "/{module:.+}/@v/{version}.info"
const pathLatest = "/{module:.+}/@latest"
const pathVersionZip = "/{module:.+}/@v/{version}.zip"

func main() {
	r := mux.NewRouter()
	dp := gdp.New("")
	r.HandleFunc(pathList, func(w http.ResponseWriter, r *http.Request) {
		module := mux.Vars(r)["module"]
		vers, err := dp.List(r.Context(), module)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, strings.Join(vers, "\n"))
	})

	r.HandleFunc(pathVersionModule, func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		module := vars["module"]
		ver := vars["version"]
		bts, err := dp.GoMod(r.Context(), module, ver)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(bts)
	})

	r.HandleFunc(pathVersionInfo, func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		module := vars["module"]
		ver := vars["version"]
		info, err := dp.Info(r.Context(), module, ver)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(info)
	})

	r.HandleFunc(pathLatest, func(w http.ResponseWriter, r *http.Request) {
		module := mux.Vars(r)["module"]
		info, err := dp.Latest(r.Context(), module)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("nice %+v\n", info)

		json.NewEncoder(w).Encode(info)
	})

	r.HandleFunc(pathVersionZip, func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		module := vars["module"]
		ver := vars["version"]
		rdr, err := dp.Zip(r.Context(), module, ver)
		if err != nil {
			fmt.Println("uhh", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
