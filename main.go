package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/stephandollberg/go-json-rest-middleware-jwt"
)

const (
	userID   = "admin"
	password = "admin"
)

func main() {
	var port string
	flag.StringVar(&port, "port", "9999", "port to listen")
	flag.Parse()

	log.Println("Starting server on localhost:" + port)

	countries := Countries{
		Store: map[string]*Country{},
	}

	// JSON Web Tokens middleware
	jwtMW := &jwt.JWTMiddleware{
		Key:        []byte("secret key"),
		Realm:      "jwt auth",
		Timeout:    time.Hour,
		MaxRefresh: time.Hour * 24,
		Authenticator: func(userid string, passwd string) bool {
			return userid == userID && passwd == password
		},
	}

	api := rest.NewApi()
	statusMW := &rest.StatusMiddleware{}
	// StatusMiddleware must be registered in api.Use() BEFORE rest.DefaultDevStack
	// because of an implicit dependency on RecorderMiddleware (request.ENV["STATUS_CODE"])
	// and reverse call order of Middleware#MiddlewareFunc().
	api.Use(statusMW)
	api.Use(rest.DefaultDevStack...)
	// we use the IfMiddleware to remove certain paths from needing authentication
	//api.Use(&rest.IfMiddleware{
	//	Condition: func(r *rest.Request) bool {
	//		return r.URL.Path != "/login"
	//	},
	//	IfTrue: jwtMW,
	//})

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

		rest.Post("/login", jwtMW.LoginHandler),
		rest.Get("/auth_test", handleAuth),
		rest.Get("/refresh_token", jwtMW.RefreshHandler),

		rest.Get("/stream", StreamThings),
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

func handleAuth(w rest.ResponseWriter, r *rest.Request) {
	returnJSON(w, map[string]string{"authed": r.Env["REMOTE_USER"].(string)})
}

// Thing is a thing.
type Thing struct {
	Name string
}

// StreamThings returns stream things.
func StreamThings(w rest.ResponseWriter, r *rest.Request) {
	cpt := 0
	for {
		cpt++
		returnJSON(w, &Thing{Name: fmt.Sprintf("thing #%d", cpt)})
		if _, e := w.(http.ResponseWriter).Write([]byte("\n")); e != nil {
			log.Println(e)
		}
		w.(http.Flusher).Flush()
		time.Sleep(time.Duration(3) * time.Second)
	}
}
