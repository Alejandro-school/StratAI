package main

import (
	"log"
	"net/http"

	"cs2-demo-service/api"
	"cs2-demo-service/middlewares"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	// Cargar el fichero .env desde la raíz del proyecto
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("No se pudo cargar el fichero .env (no es crítico):", err)
	}

	// Crea el router.
	router := mux.NewRouter()

	// Endpoint simplificado: procesa una demo y devuelve el JSON directamente
	router.HandleFunc("/process-demo", api.HandleProcessDemo).Methods("POST")
	router.HandleFunc("/health", api.HandleHealth).Methods("GET")

	// Endpoint para obtener detalles de un match desde exports/
	router.HandleFunc("/match-details/{matchID}", api.HandleGetMatchDetails).Methods("GET")

	// Aplica el middleware de CORS.
	handlerWithCors := middlewares.WithCors(router)

	log.Println("🚀 Servicio de análisis de demos CS2 iniciado en puerto :8080")
	http.ListenAndServe(":8080", handlerWithCors)
}
