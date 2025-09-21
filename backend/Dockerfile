# Usa una imagen base de Go para construir la aplicación
FROM golang:1.25-alpine AS builder

# Establece el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copia los archivos de configuración de Go Modules
COPY go.mod go.sum ./
# Descarga las dependencias (para aprovechar el caché de capas)
RUN go mod download

# Copia el código fuente de tu aplicación Go
COPY . .

# Compila la aplicación Go
# La opción -o main especifica el nombre del ejecutable final
RUN go build -o main .

# --- Segunda etapa: Usar una imagen base más pequeña para el ejecutable final ---
# Esto es un "multi-stage build" para reducir el tamaño final de la imagen
FROM alpine:latest

# Establece el directorio de trabajo para el ejecutable final
WORKDIR /root/

# Copia el ejecutable compilado desde la etapa "builder"
COPY --from=builder /app/main .

# Expone el puerto que tu aplicación Go escuchará
EXPOSE 3000

# Define el comando que se ejecutará cuando el contenedor se inicie
CMD ["./main"]
