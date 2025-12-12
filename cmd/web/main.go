package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	// Obtener directorio base del proyecto
	baseDir := "."

	// Servir archivos estáticos de /web
	webHandler := http.FileServer(http.Dir(baseDir + "/web"))
	http.Handle("/", webHandler)

	// Servir data/output para el JSON del horario
	dataHandler := http.FileServer(http.Dir(baseDir + "/data"))
	http.Handle("/data/", http.StripPrefix("/data/", dataHandler))

	fmt.Printf("🌐 Servidor iniciado en http://localhost:%s\n", port)
	fmt.Println("   Abre esta URL en tu navegador para ver el visualizador de horarios")
	fmt.Println("   Presiona Ctrl+C para detener el servidor")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
