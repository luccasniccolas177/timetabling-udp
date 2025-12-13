Proyecto final para Grafos y algoritmos: Optimización y asignación de horarios universitarios UDP - FIC

Algoritmos utilizados:

-Greedy Graph Coloring.
-Best-Fit Room Assignment.
-Simulated Annealing.
-Detección de Cliques.

Como ejecutar el proyecto:

-Es necesario tener instalado Golang.

-Una vez tengas Golang instalado en tu PC, ejecutaras el comando go build -o bin/timetabling ./cmd/api/./bin/timetabling, esto ejecutará el proyecto,
generando un archivo llamado schedule.json que incluye la asignación de la sala y el bloque de tiempo para cada actividad académica.

-Finalmente, para poder visualizar de mejor manera los datos del archivo schedule.json, ejecutaras el siguiente comando 
go build -o bin/web_server ./cmd/web./bin/web_server 3000, con esto ya hecho, iras a tu navegador y escribiras en tu url http://localhost:3000 y ya podrás visualizar el contenido del .json.


