# Deploy — Template (back + front)

Configuración de despliegue para el proyecto **template**.

```
template/
├── back/            # API Go (chi) — puerto 8080
├── front/           # Angular 21 (standalone) — build /template/
└── deploy/
    ├── nginx.conf   # server block para nginx (puerto 5000)
    └── README.md
```

---

## 1. Entorno del frontend (`front/src/environments/environment.ts`)

Según el modo en que estés corriendo, usa una u otra configuración:

### Desarrollo (local)

```ts
export const environment = {
  API_URL: 'http://localhost:8080', // Go API directa (CORS → localhost:4200)
  MEDIA_URL: 'http://localhost:5000/template/public', // estáticos públicos
};
```

> La API se consume directo en `localhost:8080` (sin pasar por nginx). Lo
> público (archivos `publico` e imágenes de entidades activas) lo sirve nginx
> desde `PUBLIC_DIR/public`; lo restringido lo sirve la API.

### Producción / Deploy (IP absoluta del servidor)

```ts
export const environment = {
  API_URL: 'http://192.168.90.68:5000/template/api',     // proxy nginx → Go API
  MEDIA_URL: 'http://192.168.90.68:5000/template/public', // nginx estáticos
};
```

> En deploy **todo** pasa por nginx (puerto 5000): el frontend, los archivos públicos
> y la API vía `/template/api/` (nginx hace proxy a `127.0.0.1:8080`).
> Cambia `192.168.90.68` por la IP real del servidor.

---

## 2. Backend (Go)

```powershell
cd back
go build -o server.exe ./cmd/server
```

Variables de entorno (`back/.env`) mínimas:

```
SERVER_PORT=:8080
DB_TYPE=sqlite          # sqlite | postgres
DB_PATH=data/template.db
PUBLIC_DIR=C:\Andriw 2\nginx-1.28.0\html\template
```

Para usar PostgreSQL:

```
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=
DB_NAME=template
DB_SSLMODE=disable
```

> Al arrancar, el backend crea las tablas si no existen y siembra los permisos
> base y el usuario `admin` / `REDACTED_DEFAULT_ADMIN_PASSWORD`. No hay modo mock.
>
> `PUBLIC_DIR` llega **hasta `template`**. Lo público vive en
> `PUBLIC_DIR/public` y lo sirve nginx:
>
> - Imágenes de entidades activas → `public/imagenes`.
> - Archivos `publico` → `public/archivos`.
>
> Lo restringido vive en `PUBLIC_DIR/archivos`, que nginx bloquea, y lo sirve
> Go:
>
> - Imágenes de entidades inactivas → `archivos/imagenes`.
> - Archivos `autenticado`/`propietario` → `archivos`.
>
> El servidor crea todas las carpetas automáticamente y mueve los archivos al
> cambiar la visibilidad o el estado de la entidad.

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

### Rutas servidas (puerto 5000)

| Ruta                        | Origen                                     |
| --------------------------- | ------------------------------------------ |
| `/template/`                | Frontend (`html/template`)                 |
| `/template/public/`         | Estáticos (`html/template/public/`)        |
| `/template/public/imagenes/`| Imágenes públicas (entidades activas)      |
| `/template/public/archivos/`| Archivos públicos (`visibilidad=publico`)  |
| `/template/api/`            | Proxy → `127.0.0.1:8080` (Go API)          |

> `proxy_pass http://127.0.0.1:8080/;` (con `/` final) hace que
> `/template/api/auth/token` llegue al backend como `/auth/token`, tal como están
> registradas las rutas del router Go.

---

## Flujo resumido

| Modo    | API_URL                     | MEDIA_URL                              | Imágenes activas / públicas          | Imágenes restringidas              |
| ------- | --------------------------- | -------------------------------------- | ------------------------------------ | ---------------------------------- |
| Dev     | `http://localhost:8080`     | `http://localhost:5000/template/public`| `PUBLIC_DIR\public\imagenes`         | `PUBLIC_DIR\archivos\imagenes`     |
| Deploy  | `http://IP:5000/template/api` | `http://IP:5000/template/public`      | `PUBLIC_DIR\public\imagenes`         | `PUBLIC_DIR\archivos\imagenes`     |

> Archivos: los `publico` van a `PUBLIC_DIR\public\archivos` (nginx); los
> `autenticado`/`propietario` a `PUBLIC_DIR\archivos` (API).
