package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq" // Driver para PostgreSQL
)

// Transaction representa una transacción de dinero
type Transaction struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"` // "income" o "expense"
	CreatedAt   time.Time `json:"created_at"`
}

var db *sql.DB

func main() {
	// Obtener variables de entorno
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	apiPort := os.Getenv("API_PORT")

	if apiPort == "" {
		apiPort = "3000" // Puerto por defecto si no se especifica
	}

	// Cadena de conexión a PostgreSQL
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	// Intentar conectar a la base de datos con reintentos
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping() // Verificar la conexión
			if err == nil {
				log.Println("Conectado a la base de datos PostgreSQL")
				break
			}
		}
		log.Printf("No se pudo conectar a la base de datos. Reintentando en 5 segundos... (%d/10)", i+1)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		log.Fatalf("Error crítico al conectar a la base de datos: %v", err)
	}
	defer db.Close()

	// Crear la tabla si no existe
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS transactions (
		id SERIAL PRIMARY KEY,
		description TEXT NOT NULL,
		amount NUMERIC(10, 2) NOT NULL,
		type VARCHAR(10) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Error al crear la tabla de transacciones: %v", err)
	}
	log.Println("Tabla 'transactions' verificada/creada.")

	// Configurar CORS (para permitir peticiones desde el frontend)
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Lista de orígenes permitidos
			allowedOrigins := []string{
				"http://165.22.139.71:8080",
				"http://localhost:8080",
				"http://127.0.0.1:8080",
			}

			// Verificar si el origen de la request está permitido
			origin := r.Header.Get("Origin")
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h.ServeHTTP(w, r)
		})
	}

	// Rutas de la API
	http.Handle("/transactions", corsHandler(http.HandlerFunc(getTransactions)))
	http.Handle("/transaction", corsHandler(http.HandlerFunc(createTransaction)))
	http.Handle("/transaction/", corsHandler(http.HandlerFunc(handleTransactionByID))) // Para PUT y DELETE

	log.Printf("Servidor backend Go escuchando en el puerto :%s", apiPort)
	log.Fatal(http.ListenAndServe(":"+apiPort, nil))
}

// Handler para /transactions (GET: obtener todas)
func getTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT id, description, amount, type, created_at FROM transactions ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Description, &t.Amount, &t.Type, &t.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		transactions = append(transactions, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}

// Handler para /transaction (POST: crear una nueva)
func createTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var t Transaction
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validación básica
	if t.Description == "" || t.Amount <= 0 || (t.Type != "income" && t.Type != "expense") {
		http.Error(w, "Descripción, monto o tipo inválido", http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO transactions(description, amount, type) VALUES($1, $2, $3) RETURNING id, created_at")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(t.Description, t.Amount, t.Type).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// Handler genérico para /transaction/{id} (PUT: actualizar, DELETE: borrar)
func handleTransactionByID(w http.ResponseWriter, r *http.Request) {
	// Extraer ID de la URL
	pathParts := splitPath(r.URL.Path)
	if len(pathParts) < 2 {
		http.Error(w, "ID de transacción no proporcionado", http.StatusBadRequest)
		return
	}
	idStr := pathParts[len(pathParts)-1] // Última parte de la URL
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID de transacción inválido", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "PUT":
		updateTransaction(w, r, id)
	case "DELETE":
		deleteTransaction(w, r, id)
	case "GET": // Opcional: obtener una sola transacción por ID
		getTransactionByID(w, r, id)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range strings.Split(path, string(os.PathSeparator)) {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// Handler para /transaction/{id} (GET: obtener por ID)
func getTransactionByID(w http.ResponseWriter, r *http.Request, id int) {
	row := db.QueryRow("SELECT id, description, amount, type, created_at FROM transactions WHERE id = $1", id)

	var t Transaction
	err := row.Scan(&t.ID, &t.Description, &t.Amount, &t.Type, &t.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Transacción no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// Handler para /transaction/{id} (PUT: actualizar)
func updateTransaction(w http.ResponseWriter, r *http.Request, id int) {
	var t Transaction
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validación básica
	if t.Description == "" || t.Amount <= 0 || (t.Type != "income" && t.Type != "expense") {
		http.Error(w, "Descripción, monto o tipo inválido", http.StatusBadRequest)
		return
	}

	res, err := db.Exec("UPDATE transactions SET description=$1, amount=$2, type=$3 WHERE id=$4",
		t.Description, t.Amount, t.Type, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Transacción no encontrada", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Transacción %d actualizada correctamente", id)
}

// Handler para /transaction/{id} (DELETE: borrar)
func deleteTransaction(w http.ResponseWriter, r *http.Request, id int) {
	res, err := db.Exec("DELETE FROM transactions WHERE id=$1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Transacción no encontrada", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Transacción %d eliminada correctamente", id)
}
