# Firebase Setup Guide

## Firebase Configuration

Project ID: `jetlink-47eb8`

### Setup Steps

#### 1. **Enable Authentication in Firebase Console**

1. Go to [Firebase Console](https://console.firebase.google.com/project/jetlink-47eb8)
2. Navigate to **Authentication** → **Sign-in method**
3. Enable **Google** sign-in provider
4. Add your authorized domains (e.g., `localhost` for development)

#### 2. **Get Service Account Key (Optional - for production)**

For production deployment, you'll need a service account key:

1. Go to **Project Settings** → **Service Accounts**
2. Click **Generate New Private Key**
3. Save the JSON file as `serviceAccountKey.json` in the backend directory
4. Update `.env`:
   ```
   FIREBASE_SERVICE_ACCOUNT_KEY=./serviceAccountKey.json
   ```

#### 3. **Development Mode (No Service Account Key)**

For development, Firebase Admin SDK will use Application Default Credentials (ADC):

```bash
# Install gcloud CLI if not already installed
# Login with your Google account
gcloud auth application-default login

# Set the project
gcloud config set project jetlink-47eb8
```

The SDK will automatically authenticate using your logged-in account.

#### 4. **Update Environment Variables**

Copy `.env.example` to `.env`:

```bash
cp .env.example .env
```

Update the following variables:

```env
# Firebase
FIREBASE_PROJECT_ID=jetlink-47eb8

# Database
MYSQL_DSN=user:password@tcp(localhost:3306)/jetlink?charset=utf8mb4&parseTime=True&loc=Local

# Server
SERVER_ADDR=:8080
```

#### 5. **Run Backend**

```bash
go run cmd/server/main.go
```

You should see:
```
INFO: Firebase Admin SDK initialized successfully
INFO: Connected to MySQL...
INFO: Connected to Redis...
```

## Testing

### Test Token Verification

1. **Get Firebase ID Token from Frontend:**
   ```javascript
   const user = firebase.auth().currentUser;
   const token = await user.getIdToken();
   console.log(token);
   ```

2. **Test API with Token:**
   ```bash
   curl -X GET http://localhost:8080/api/auth/driver-status \
     -H "Authorization: Bearer <your-firebase-token>"
   ```

3. **Expected Response:**
   ```json
   {
     "success": true,
     "data": {
       "isDriver": true,
       "isVerified": true,
       "vehicleType": "Toyota Avanza",
       "vehiclePlate": "B 1234 ABC",
       ...
     }
   }
   ```

## Troubleshooting

### Error: "Firebase initialization failed"

- Make sure you have run `gcloud auth application-default login`
- Check that your Google account has access to the Firebase project
- Verify `FIREBASE_PROJECT_ID` is correct in `.env`

### Error: "Invalid token"

- Make sure the token is not expired (tokens expire after 1 hour)
- Check that Firebase Authentication is enabled in the Firebase Console
- Verify the token is from the correct Firebase project

### Error: "Missing Authorization header"

- Make sure you're sending the token in the Authorization header
- Format should be: `Authorization: Bearer <token>`
