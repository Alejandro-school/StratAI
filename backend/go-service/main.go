package main

import (
	"log"
	"net/http"

	// Importamos nuestros paquetes locales
	"cs2-demo-service/db" // si creaste db/redis.go
	"cs2-demo-service/handlers"

	//"cs2-demo-service/inspect"
	"cs2-demo-service/middlewares"

	"github.com/gorilla/mux"
)

func main() {
	// Inicializa Redis (podrías hacerlo aquí o en un init() dentro de db/redis.go)
	db.InitRedis()

	// Llamamos alguna función de inspección si la necesitas
	//inspect.InspectDemo()

	// Creamos el router de Gorilla
	router := mux.NewRouter()

	// Registramos los endpoints y sus handlers
	router.HandleFunc("/process-downloaded-demo", handlers.HandleProcessDownloadedDemo).Methods("POST")
	router.HandleFunc("/get-processed-demos", handlers.HandleGetProcessedDemos).Methods("GET")
	router.HandleFunc("/match/{steamID}/{matchID}", handlers.HandleGetMatchByID).Methods("GET")

	// Aplicamos el middleware de CORS
	handlerWithCors := middlewares.WithCors(router)

	// Arrancamos el servidor
	log.Println("Servidor corriendo en puerto :8080")
	http.ListenAndServe(":8080", handlerWithCors)
}
