

# GO CDN STACK

Este proyecto implementa un sistema de **Red de Distribución de Contenido (CDN)** utilizando **Go (Golang)** con las siguientes tecnologías:

- **Gin** (framework HTTP)
- **MongoDB** (persistencia del panel de control)
- **NATS** (bus de mensajería para eventos y verificaciones de estado)
- Estructura **Monorepo** con múltiples servicios

## 📂 Estructura del Repositorio

```
.
├── control-panel       # Management API + MongoDB persistence + NATS subscriber
├── edge                # Edge CDN service (serves cached content, proxies requests)
├── mid                 # Mid-tier service (cache logic, sync with control panel, health checks)
└── origin-sample       # Sample origin server (static files for testing)
```

### Desglose de Servicios

#### **Panel de Control**
- API REST para gestionar usuarios, CDNs e instantáneas (snapshots)
- Persiste datos en MongoDB
- Se suscribe a actualizaciones de estado de los servicios a través de NATS

#### **Edge**
- Recibe solicitudes del cliente
- Almacena en caché contenido estático (imágenes, CSS, JS, fuentes, video, audio)
- Reenvía solicitudes dinámicas o no almacenables en caché a los servidores de origen
- Almacena la caché en disco

#### **Mid**
- Sincroniza configuraciones de CDN desde el panel de control (a través de la API de instantáneas)
- Almacena en caché respuestas localmente (en disco + metadatos en memoria)
- Publica el estado de salud del servicio a través de NATS
- Se suscribe a actualizaciones de instantáneas desde el panel de control

#### **Muestra de Origen**
- Servidor de archivos estáticos simple (imágenes, JSON, video) para probar flujos de CDN

---

## 🚀 Cómo Ejecutarlo

### 1. Clonar el repositorio

```bash
git clone https://github.com/AmirAghaee/go-cdn-stack.git
cd go-cdn-stack
```

### 2. Ejecutar el Panel de Control

Asegúrate de que MongoDB y NATS estén en ejecución.

```bash
cd control-panel
go run main.go
```

### 3. Ejecutar el Servicio Mid

```bash
cd mid
go run main.go
```

### 4. Ejecutar el Servicio Edge

```bash
cd edge
go run main.go
```

### 5. Ejecutar la Muestra de Origen

```bash
cd origin-sample
go run main.go
```

---

## ⚡ Notas de Desarrollo

- Reglas de caché: Solo se almacenan en caché las respuestas de tipo `image/*`, `font/*`, `text/css`, `text/javascript`, `application/javascript`, `video/*` y `audio/*`.
- Las solicitudes que no son GET se reenvían directamente al origen.
- Cada elemento en caché tiene metadatos almacenados junto al archivo en caché (encabezados + tiempo de expiración).
- El nivel intermedio (Mid) sincroniza las CDNs desde el Panel de Control al inicio y también a través de eventos de NATS.
- Los mensajes de verificación de estado son publicados por los servicios y consumidos por el Panel de Control.

---

## 🛠️ Tecnologías Utilizadas

- **Lenguaje:** Go 1.25+
- **Frameworks:** Gin, NATS, controlador de MongoDB
- **Persistencia:** MongoDB (panel de control), caché basada en disco (edge/mid)

---

## 📌 PENDIENTE (TODO)

- [ ] Agregar Docker Compose para desarrollo local (MongoDB + NATS + servicios)
- [ ] Agregar pruebas unitarias/de integración
- [ ] Implementar invalidación de caché a través de NATS
- [ ] Agregar middleware de límite de tasa y registro de logs

---
