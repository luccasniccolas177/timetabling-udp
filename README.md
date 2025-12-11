# UDP Timetabling Solver

Sistema automatizado de programación de horarios universitarios escrito en **Go**. Este proyecto utiliza algoritmos avanzados de grafos y metaheurísticas para generar horarios factibles y optimizados, respetando tanto restricciones duras (hard constraints) como preferencias de negocio (soft constraints).

## 🚀 Características Principales

### 🎯 Algoritmos Implementados
El solver utiliza un enfoque híbrido de tres fases:

1.  **Fase Constructiva (Greedy Graph Coloring):**
    *   Construye una solución inicial factible coloreando un **Grafo de Conflictos**.
    *   Nodos = Sesiones de Clases.
    *   Aristas = Conflictos (Mismo Profesor, Misma Sala Preasignada, Mismo Nivel/Semestre).
    *   Usa heurísticas de saturación (DSATUR-like) para ordenar la asignación.

2.  **Fase de Optimización (Simulated Annealing):**
    *   Mejora la solución inicial iterativamente (50,000 iteraciones).
    *   **Función de Costo:** Penaliza violaciones a preferencias suaves.
    *   **Búsqueda Inteligente (Smart Move):** Detecta automáticamente "sesiones hermanas" e intenta moverlas a bloques espejo o con separación ideal.

3.  **Asignación de Salas (Burke et al.):**
    *   Asigna salas físicas a las clases basándose en la capacidad y restricciones.
    *   Implementa priorización de *"Misma Sala"* para mantener consistencia en un mismo curso.
    *   Maneja re-coloreo (desplazamiento) si no hay salas disponibles en un bloque.

### ✅ Reglas de Negocio (Constraints)

#### Restricciones Duras (Hard Constraints)
*   **Conflictos de Profesor:** Un profesor no puede estar en dos lugares a la vez.
*   **Conflictos de Curso:** Sesiones del mismo curso/sección no pueden toparse.
*   **Conflictos de Nivel:** Cursos del mismo semestre (malla) no deben toparse (para permitir tomarlos todos).
*   **Capacidad de Sala:** El curso no puede exceder el tamaño de la sala.

#### Restricciones Suaves y Preferencias (Soft Constraints)
*   **Horarios Espejo:** Las cátedras de una misma sección deben tener el mismo horario en días distintos (ej. 10:00).
*   **Separación de Días (Gap):** Se prioriza fuertemente una separación de **3 días** (Lunes-Jueves, Martes-Viernes).
*   **Misma Sala:** Se intenta asignar la misma sala física para todas las cátedras de una sección.
*   **Ayudantías en Miércoles:** Se prioriza agendar ayudantías en el bloque de la tarde de los miércoles.
*   **Balanceo de Carga:** Distribución equitativa de clases para evitar saturación de pasillos/recursos.

## 🛠️ Estructura del Proyecto

*   `cmd/api/main.go`: Punto de entrada. Orquesta la carga de datos, construcción del grafo, y ejecución de los solvers.
*   `internal/domain/`: Definiciones de structs (Curso, Sección, Sala, Profesor).
*   `internal/graph/`: Lógica del grafo de conflictos.
*   `internal/solver/coloring.go`: Algoritmo Greedy de coloreo.
*   `internal/solver/simulated_annealing.go`: Motor de optimización SA.
*   `internal/solver/burke_room_assignment.go`: Algoritmo de asignación de salas.

## 💻 Ejecución

Para correr el generador de horarios:

```bash
go run cmd/api/main.go
```

Esto generará un archivo `horario_detalle.json` con el cronograma completo estructurado por curso y sección.

## 📊 Resultados Típicos

El sistema logra reducir la carga máxima por bloque de ~80 sesiones (aleatorio) a ~35-40 (balanceado), logrando alineación de horarios espejo en la mayoría de los casos factibles (Gap de 3 días) y manteniendo la consistencia de salas.
