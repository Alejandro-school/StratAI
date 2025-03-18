package main

import (
	"log"
	"net/http"

	"cs2-demo-service/db"
	"cs2-demo-service/handlers"
	"cs2-demo-service/middlewares"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	// Cargar el fichero .env desde la raíz del backend
	err := godotenv.Load("../.env") // Ajusta la ruta según la ubicación de tu .env
	if err != nil {
		log.Println("No se pudo cargar el fichero .env:", err)
	}
	// Inicializa Redis.
	db.InitRedis()

	// Crea el router.
	router := mux.NewRouter()

	// Registra los endpoints.
	router.HandleFunc("/process-downloaded-demo", handlers.HandleProcessDownloadedDemo).Methods("POST")
	router.HandleFunc("/get-processed-demos", handlers.HandleGetProcessedDemos).Methods("GET")
	router.HandleFunc("/match/{steamID}/{matchID}", handlers.HandleGetMatchByID).Methods("GET")

	// Aplica el middleware de CORS.
	handlerWithCors := middlewares.WithCors(router)

	log.Println("Servidor corriendo en puerto :8080")
	http.ListenAndServe(":8080", handlerWithCors)
}
