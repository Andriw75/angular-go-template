# Deploy — Template (back + front)

Configuración de despliegue para el proyecto **template**.

```
template/
├── back/            # API Go (chi) — puerto 8080
├── front/           # Angular 21 (standalone) — build /template/
└── deploy/
    ├── nginx.conf   # server block para nginx (puerto 4000)
    └── README.md
```

---

## 1. Entorno del frontend (`front/src/environments/environment.ts`)

Según el modo en que estés corriendo, usa una u otra configuración:

### Desarrollo (local)

```ts
export const environment = {
  API_URL: 'http://localhost:8080', // Go API directa (CORS → localhost:4200)
  MEDIA_URL: 'http://localhost:4000/template/public', // nginx sirve las imágenes
};
```

> La API se consume directo en `localhost:8080` (sin pasar por nginx). Las imágenes
> sí las sirve nginx porque el backend escribe en `PUBLIC_DIR`.

### Producción / Deploy (IP absoluta del servidor)

```ts
export const environment = {
  API_URL: 'http://192.168.90.68:4000/template/api',     // proxy nginx → Go API
  MEDIA_URL: 'http://192.168.90.68:4000/template/public', // nginx estáticos
};
```

> En deploy **todo** pasa por nginx (puerto 4000): el frontend, los archivos públicos
> y la API vía `/template/api/` (nginx hace proxy a `127.0.0.1:8080`).
> Cambia `192.168.90.68` por la IP real del servidor.

---

## 2. Backend (Go)

```powershell
cd back
go build -o server.exe ./cmd/server
```

Variables de entorno (`back/.env`) mínimas para imágenes:

```
SERVER_PORT=:8080
USE_MOCK=true
PUBLIC_DIR=C:\Andriw 2\nginx-1.28.0\html\template
```

> `PUBLIC_DIR` llega **hasta `template`**; el código agrega `/public` (y dentro
> `/imagenes`) de forma fija, dejando espacio a futuros `private`, `user`, etc.
> El servidor crea `public/imagenes` automáticamente si no existe.

Ejecutar:

```powershell
.\server.exe
```

---

## 3. Frontend (Angular)

```powershell
cd front
npm install
# build con base-href para que funcione bajo /template/
npm run build -- --base-href=/template/
```

Copiar el resultado a la carpeta que sirve nginx:

```powershell
# el build genera front/dist/front
Copy-Item -Recurse -Force .\dist\front\* "C:\Andriw 2\nginx-1.28.0\html\template\"
```

> Asegúrate de que `front/src/environments/environment.ts` tenga la configuración
> correcta (desarrollo o deploy) **antes** de compilar.

---

## 4. nginx

1. Copia el contenido de `deploy/nginx.conf` dentro del bloque `http { ... }` de tu
   `nginx.conf` (o en un include).
2. Recarga la configuración:

```powershell
nginx -s reload
```

### Rutas servidas (puerto 4000)

| Ruta                        | Origen                                     |
| --------------------------- | ------------------------------------------ |
| `/template/`                | Frontend (`html/template`)                 |
| `/template/public/`         | Estáticos (`html/template/public/`)        |
| `/template/public/imagenes/`| Imágenes de categorías y productos         |
| `/template/api/`            | Proxy → `127.0.0.1:8080` (Go API)          |

> `proxy_pass http://127.0.0.1:8080/;` (con `/` final) hace que
> `/template/api/auth/token` llegue al backend como `/auth/token`, tal como están
> registradas las rutas del router Go.

---

## Flujo resumido

| Modo    | API_URL                     | MEDIA_URL                              | Imágenes escritas en            |
| ------- | --------------------------- | -------------------------------------- | ------------------------------- |
| Dev     | `http://localhost:8080`     | `http://localhost:4000/template/public`| `PUBLIC_DIR\public\imagenes`    |
| Deploy  | `http://IP:4000/template/api` | `http://IP:4000/template/public`      | `PUBLIC_DIR\public\imagenes`    |
