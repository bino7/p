package p

import (
	"github.com/gorilla/websocket"
	"net/http"
	"github.com/rs/cors"

)

var upgrader = websocket.Upgrader{
	CheckOrigin:func(r *http.Request) bool {
		return true
	},
}

var frontendServer string

func Run(){
	initStore()
	mux := http.NewServeMux()
	mux.HandleFunc("/auth",auth)
	mux.HandleFunc("/register",register)
	mux.Handle("/peer",newPeer())

	c := cors.New(cors.Options{
		AllowedOrigins:     []string{"http://localhost:8080"},
		AllowedMethods:     []string{"POST", "GET", "PUT", "DELETE"},
		AllowedHeaders:     []string{"Content-Type","Content-Range","Content-Disposition",
						"Accept","Authorization","Set-Cookie",},
		ExposedHeaders:     []string{"Content-Type","Content-Range","Content-Disposition",
						"Accept","Authorization","Set-Cookie",},
		AllowCredentials:   true,
	})

	http.ListenAndServe(":3000", c.Handler(mux))
}