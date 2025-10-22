
---

# 🧠 YOLO Server (Go + Python + PostgreSQL)

Server ini merupakan backend modular untuk sistem deteksi dan perbandingan BOM menggunakan **Go**, **Python FastAPI**, dan **PostgreSQL**.
Arsitektur menggunakan Docker Compose agar setiap komponen bisa dijalankan secara terpisah dan mudah di-deploy.

---

## 📁 Struktur Project

```
yolo-server/
│
├── main.go
├── Dockerfile
├── docker-compose.yml
├── db/
├── handlers/
│   ├── action/
│   ├── bom/
│   ├── detection/
│   └── routes.go
├── models/
│   └── models.go
├── python_predictor/
│   └── Dockerfile
├── uploads/
├── init.sql
├── .env.dev
└── .env.prod
```

---

## ⚙️ 1. Persyaratan Awal

Pastikan kamu sudah install:

* 🐳 [Docker](https://www.docker.com/)
* 🧱 [Docker Compose](https://docs.docker.com/compose/install/)
* 🐹 (opsional) Go 1.25+ kalau mau run tanpa Docker

---

## 🔑 2. File Environment

Ada dua konfigurasi environment:

### 🔹 `.env.dev` (untuk development)

```
POSTGRES_HOST=postgres
POSTGRES_USER=user
POSTGRES_PASSWORD=password
POSTGRES_DB=yolo_db
PYTHON_API_URL=http://python-api:5001/predict
PORT=8081
POSTGRES_PORT=5433
PYTHON_PORT=5001
```

### 🔹 `.env.prod` (untuk deployment)

```
POSTGRES_HOST=postgres
POSTGRES_USER=prod_user
POSTGRES_PASSWORD=prod_password
POSTGRES_DB=yolo_db
PYTHON_API_URL=http://python-api:5001/predict
PORT=80
POSTGRES_PORT=5432
PYTHON_PORT=5001
```

---

## 🚀 3. Menjalankan Project

### ▶️ Mode **Development**

Jalankan service dengan file `.env.dev`:

```bash
ENV_FILE=.env.dev docker compose up --build
```

Aplikasi Go akan berjalan di:

```
http://localhost:8081
```

Database bisa diakses di:

```
localhost:5433
```

Python API:

```
http://localhost:5001
```

---

### ⚡ Mode **Production**

Gunakan file environment untuk production:

```bash
ENV_FILE=.env.prod docker compose up --build -d
```

Akan berjalan di port 80, bisa diakses di:

```
http://localhost
```

---

## 🧩 4. Struktur Service (Docker Compose)

| Service      | Port Host | Container | Deskripsi                        |
| ------------ | --------- | --------- | -------------------------------- |
| `go-api`     | `8081`    | `8080`    | Backend utama (Gin + PostgreSQL) |
| `python-api` | `5001`    | `5001`    | Model YOLO / deteksi objek       |
| `postgres`   | `5433`    | `5432`    | Database PostgreSQL              |

---

## 📦 5. Build Manual (opsional, tanpa Docker)

Kalau ingin menjalankan Go API langsung dari terminal:

```bash
go mod tidy
go run main.go
```

Pastikan database PostgreSQL sudah aktif (misalnya via `docker run postgres`)
dan file `.env.dev` sudah ada di root.

---

## 🧰 6. Perintah Penting

| Perintah                                                         | Fungsi                         |
| ---------------------------------------------------------------- | ------------------------------ |
| `docker compose up --build`                                      | Build & jalankan semua service |
| `docker compose down`                                            | Hentikan semua container       |
| `docker compose logs -f go-api`                                  | Lihat log backend Go           |
| `docker exec -it yolo-server-postgres-1 psql -U user -d yolo_db` | Masuk ke database              |

---

## 🧠 7. Arsitektur Singkat

```
+-------------+         +------------------+         +-------------+
|   Frontend  | <-----> |   Go API (Gin)   | <-----> |  PostgreSQL |
| (Flutter/Web)|        |   :8080 container |        |   :5432      |
+-------------+         +------------------+         +-------------+
        |
        v
 +------------------+
 | Python API (YOLO)|
 | :5001 container  |
 +------------------+
```

---