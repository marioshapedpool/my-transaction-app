# Primera etapa: Construir la aplicación React con Node.js
FROM node:18-alpine AS builder

# Establece el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copia los archivos de configuración de Node.js
COPY package.json package-lock.json ./

# Instala las dependencias
RUN npm install

# Copia el código fuente del frontend
COPY . .

# Compila la aplicación React para producción
# El comando 'build' de Vite por defecto genera los archivos en la carpeta 'dist'
RUN npm run build

# --- Segunda etapa: Usar una imagen de Nginx para servir los archivos estáticos ---
# Nginx es un servidor web muy ligero y eficiente para servir archivos estáticos
FROM nginx:alpine

# Copia los archivos de configuración personalizados de Nginx
# Asegúrate de que el path sea correcto si modificas el archivo default.conf
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copia los archivos estáticos compilados de tu aplicación React desde la etapa 'builder'
COPY --from=builder /app/dist /usr/share/nginx/html

# Expone el puerto 80, que es el puerto HTTP estándar que Nginx escuchará
EXPOSE 80

# Define el comando que inicia Nginx cuando el contenedor se ejecute
CMD ["nginx", "-g", "daemon off;"]
