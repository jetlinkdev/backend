package firebase

import (
	"context"
	"fmt"

	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// FirebaseApp holds the Firebase app and auth client
type FirebaseApp struct {
	App  *firebase.App
	Auth *auth.Client
}

var firebaseApp *FirebaseApp

// InitFirebase initializes Firebase Admin SDK
func InitFirebase(serviceAccountKeyPath string) error {
	ctx := context.Background()

	// Initialize with service account key if provided
	var opts []option.ClientOption
	if serviceAccountKeyPath != "" {
		opts = append(opts, option.WithCredentialsFile(serviceAccountKeyPath))
	}

	// Initialize Firebase app
	app, err := firebase.NewApp(ctx, nil, opts...)
	if err != nil {
		return fmt.Errorf("error initializing firebase app: %v", err)
	}

	// Get auth client
	authClient, err := app.Auth(ctx)
	if err != nil {
		return fmt.Errorf("error getting auth client: %v", err)
	}

	firebaseApp = &FirebaseApp{
		App:  app,
		Auth: authClient,
	}

	return nil
}

// GetAuthClient returns the Firebase auth client
func GetAuthClient() (*auth.Client, error) {
	if firebaseApp == nil {
		return nil, fmt.Errorf("firebase app not initialized")
	}
	return firebaseApp.Auth, nil
}

// VerifyIDToken verifies a Firebase ID token and returns the token
func VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	if firebaseApp == nil {
		return nil, fmt.Errorf("firebase app not initialized")
	}

	token, err := firebaseApp.Auth.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("error verifying id token: %v", err)
	}

	return token, nil
}

// GetUser gets a user by UID
func GetUser(ctx context.Context, uid string) (*auth.UserRecord, error) {
	if firebaseApp == nil {
		return nil, fmt.Errorf("firebase app not initialized")
	}

	user, err := firebaseApp.Auth.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error getting user: %v", err)
	}

	return user, nil
}

// InitFirebaseWithConfig initializes Firebase using service account key or ADC
func InitFirebaseWithConfig(projectID, serviceAccountKeyPath string) error {
	ctx := context.Background()

	// Initialize with service account key if provided
	var opts []option.ClientOption
	if serviceAccountKeyPath != "" {
		opts = append(opts, option.WithCredentialsFile(serviceAccountKeyPath))
		fmt.Printf("Firebase: Using service account key: %s\n", serviceAccountKeyPath)
	} else {
		fmt.Println("Firebase: No service account key provided, using Application Default Credentials")
	}

	config := &firebase.Config{
		ProjectID: projectID,
	}

	// Initialize Firebase app
	app, err := firebase.NewApp(ctx, config, opts...)
	if err != nil {
		return fmt.Errorf("error initializing firebase app: %v", err)
	}

	// Get auth client
	authClient, err := app.Auth(ctx)
	if err != nil {
		return fmt.Errorf("error getting auth client: %v", err)
	}

	firebaseApp = &FirebaseApp{
		App:  app,
		Auth: authClient,
	}

	fmt.Println("Firebase Admin SDK initialized successfully")
	return nil
}
