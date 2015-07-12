package main

import (
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
)

func main() {
	port := flag.String("port", "9999", "port to listen")
	flag.Parse()

	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)
	router, err := rest.MakeRouter(
		rest.Get("/", func(w rest.ResponseWriter, r *rest.Request) {
			if e := w.WriteJson(map[string]string{"Body": "Hello, World!"}); e != nil {
				log.Print(e)
			}
		}),
		rest.Get("/lookup/#host", func(w rest.ResponseWriter, r *rest.Request) {
			ip, e := net.LookupIP(r.PathParam("host"))
			if e != nil {
				rest.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}
			if e := w.WriteJson(&ip); e != nil {
				log.Println(e)
			}
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	api.SetApp(router)
	log.Fatal(http.ListenAndServe(":"+*port, api.MakeHandler()))
}
