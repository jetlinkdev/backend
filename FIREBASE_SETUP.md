# Firebase Setup Guide

## ⚠️ PENTING: Kenapa Muncul Log "Firebase token verification will not be available"

Log ini muncul karena Firebase Admin SDK **tidak bisa initialize** tanpa credentials.

Backend butuh **salah satu** dari ini:
1. ✅ **Service Account Key** (RECOMMENDED)
2. ✅ **Application Default Credentials (ADC)** (untuk development)

---

## **Opsi 1: Service Account Key (RECOMMENDED)**

### **Step 1: Download Service Account Key**

1. Buka link ini: https://console.firebase.google.com/project/jetlink-47eb8/settings/serviceaccounts/adminsdk

2. Klik tombol **"Generate new private key"**

3. Klik **"Generate key"** untuk download file JSON

4. Simpan file JSON dengan nama `serviceAccountKey.json` di folder:
   ```
   /home/ferdifir/development/jet/backend/serviceAccountKey.json
   ```

### **Step 2: Buat File `.env`**

```bash
cd /home/ferdifir/development/jet/backend
cp .env.example .env
```

### **Step 3: Edit `.env`**

Buka file `.env` dan tambahkan:

```env
# Firebase Configuration
FIREBASE_PROJECT_ID=jetlink-47eb8
FIREBASE_SERVICE_ACCOUNT_KEY=./serviceAccountKey.json

# Database Configuration
MYSQL_DSN=ferdifir:WsQ4g|1N4"56@tcp(localhost:3306)/jetlink?charset=utf8mb4&parseTime=True&loc=Local

# Server Configuration
SERVER_ADDR=:8080
```

### **Step 4: Jalankan Backend**

```bash
cd /home/ferdifir/development/jet/backend
go run cmd/server/main.go
```

**Expected Output:**
```
INFO: Firebase: Using service account key: ./serviceAccountKey.json
INFO: Firebase Admin SDK initialized successfully
INFO: Connected to MySQL...
```

✅ **Sekarang token verification sudah aktif!**

---

## **Opsi 2: Application Default Credentials (Development Only)**

Gunakan ini kalau tidak mau download service account key.

### **Step 1: Install gcloud CLI**

Jika belum install:
```bash
# Ubuntu/Debian
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
gcloud init
```

### **Step 2: Login dengan Google Account**

```bash
gcloud auth application-default login
```

Browser akan terbuka, login dengan akun Google yang punya akses ke Firebase project `jetlink-47eb8`.

### **Step 3: Set Project**

```bash
gcloud config set project jetlink-47eb8
```

### **Step 4: Buat File `.env`**

```bash
cd /home/ferdifir/development/jet/backend
cp .env.example .env
```

### **Step 5: Edit `.env`**

```env
# Firebase Configuration
FIREBASE_PROJECT_ID=jetlink-47eb8
# COMMENT OUT service account key line
# FIREBASE_SERVICE_ACCOUNT_KEY=./serviceAccountKey.json

# Database Configuration
MYSQL_DSN=ferdifir:WsQ4g|1N4"56@tcp(localhost:3306)/jetlink?charset=utf8mb4&parseTime=True&loc=Local

# Server Configuration
SERVER_ADDR=:8080
```

### **Step 6: Jalankan Backend**

```bash
cd /home/ferdifir/development/jet/backend
go run cmd/server/main.go
```

**Expected Output:**
```
INFO: Firebase: No service account key provided, using Application Default Credentials
INFO: Firebase Admin SDK initialized successfully
INFO: Connected to MySQL...
```

✅ **Sekarang token verification sudah aktif!**

---

## **Testing Token Verification**

### **1. Get Firebase ID Token dari Flutter App**

Di Flutter app, setelah login dengan Google:

```dart
final user = FirebaseAuth.instance.currentUser;
final idToken = await user?.getIdToken();
print('ID Token: $idToken');
```

### **2. Test API dengan Token**

```bash
# Check driver status
curl -X GET http://localhost:8080/api/auth/driver-status \
  -H "Authorization: Bearer <your-firebase-token-here>"

# Expected Response (Success):
{
  "success": true,
  "data": {
    "isDriver": true,
    "isVerified": true,
    ...
  }
}

# Expected Response (Unauthorized):
# Jika token tidak valid atau tidak ada
```

---

## **Troubleshooting**

### **Error: "Firebase initialization failed: dial tcp: lookup oauth2.googleapis.com"**

**Problem:** Tidak ada koneksi internet atau DNS issue.

**Solution:**
- Check koneksi internet
- Restart router jika perlu

### **Error: "Firebase initialization failed: credentials: could not find default credentials"**

**Problem:** ADC tidak setup atau service account key tidak ditemukan.

**Solution:**
- **Opsi 1:** Pastikan `serviceAccountKey.json` ada di folder backend
- **Opsi 2:** Jalankan `gcloud auth application-default login`

### **Error: "Invalid token" saat test API**

**Problem:** Token expired atau tidak valid.

**Solution:**
- Token Firebase expire setelah 1 jam
- Get fresh token dari Flutter app
- Pastikan token dari project yang benar (`jetlink-47eb8`)

### **Log: "Firebase token verification will not be available"**

**Problem:** Firebase Admin SDK tidak bisa initialize.

**Solution:**
- Check `.env` file ada dan konfigurasi benar
- Check `serviceAccountKey.json` file ada (kalau pakai Opsi 1)
- Check `gcloud auth application-default login` sudah dijalankan (kalau pakai Opsi 2)

---

## **Production Deployment**

Untuk production, **WAJIB pakai Service Account Key**:

1. Download service account key dari Firebase Console
2. Simpan di secure location (bukan di git!)
3. Set environment variable di production server:
   ```bash
   export FIREBASE_SERVICE_ACCOUNT_KEY=/path/to/serviceAccountKey.json
   ```
4. Restart backend service

---

## **Security Notes**

⚠️ **JANGAN commit `serviceAccountKey.json` ke Git!**

File `.gitignore` sudah include:
```
serviceAccountKey.json
*.json
```

Tapi tetap double-check sebelum commit!

---

## **Summary**

| Method | Use Case | Security | Setup Complexity |
|--------|----------|----------|------------------|
| Service Account Key | Production | High | Easy |
| ADC (gcloud login) | Development | Medium | Very Easy |

**Recommendation:**
- **Development:** Pakai ADC (gcloud login)
- **Production:** Pakai Service Account Key
