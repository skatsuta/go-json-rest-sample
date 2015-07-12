package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/ant0ine/go-json-rest/rest"
)

func main() {
	var port string
	flag.StringVar(&port, "port", "9999", "port to listen")
	flag.Parse()

	log.Println("Starting server on localhost:" + port)

	countries := Countries{
		Store: map[string]*Country{},
	}

	api := rest.NewApi()
	statusMW := &rest.StatusMiddleware{}
	api.Use(statusMW)
	api.Use(rest.DefaultDevStack...)
	router, err := rest.MakeRouter(
		rest.Get("/", func(w rest.ResponseWriter, r *rest.Request) {
			if e := w.WriteJson(map[string]string{"Body": "Hello, World!"}); e != nil {
				log.Println(e)
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
		rest.Get("/countries", countries.GetAllCountries),
		rest.Post("/countries", countries.PostCountry),
		rest.Get("/countries/:code", countries.GetCountry),
		rest.Delete("/countries/:code", countries.DeleteCountry),
		rest.Get("/stats", func(w rest.ResponseWriter, r *rest.Request) {
			returnJSON(w, statusMW.GetStatus())
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/api/", http.StripPrefix("/api", api.MakeHandler()))
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("."))))

	api.SetApp(router)

	log.Fatal(http.ListenAndServe(":"+port, api.MakeHandler()))
}

// Country is a country.
type Country struct {
	Code string
	Name string
}

// Countries is a collection of countries.
type Countries struct {
	sync.RWMutex
	Store map[string]*Country
}

// GetCountry returns a country corresponding the given country code.
func (c *Countries) GetCountry(w rest.ResponseWriter, r *rest.Request) {
	code := r.PathParam("code")

	c.RLock()
	var country *Country
	if c.Store[code] != nil {
		country = &Country{}
		*country = *c.Store[code]
	}
	c.RUnlock()

	if country == nil {
		rest.NotFound(w, r)
		return
	}
	returnJSON(w, &country)
}

// GetAllCountries returns all countries registered.
func (c *Countries) GetAllCountries(w rest.ResponseWriter, r *rest.Request) {
	c.RLock()

	countries := make([]Country, len(c.Store))
	i := 0
	for _, country := range c.Store {
		countries[i] = *country
		i++
	}
	c.RUnlock()
	returnJSON(w, countries)
}

// PostCountry adds a posted code and countriy.
func (c *Countries) PostCountry(w rest.ResponseWriter, r *rest.Request) {
	country := Country{}
	if e := r.DecodeJsonPayload(&country); e != nil {
		rest.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}

	if country.Code == "" {
		rest.Error(w, "country code required", 400)
		return
	}

	if country.Name == "" {
		rest.Error(w, "country name required", 400)
		return
	}

	c.Lock()
	c.Store[country.Code] = &country
	c.Unlock()

	returnJSON(w, &country)
}

// DeleteCountry deletes a specified country.
func (c *Countries) DeleteCountry(w rest.ResponseWriter, r *rest.Request) {
	code := r.PathParam("code")
	c.Lock()
	delete(c.Store, code)
	c.Unlock()
	w.WriteHeader(http.StatusOK)
}

func returnJSON(w rest.ResponseWriter, v interface{}) {
	if e := w.WriteJson(v); e != nil {
		log.Println(e)
	}
}
