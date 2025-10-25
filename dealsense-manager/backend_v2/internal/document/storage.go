package document

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// StorageConfig holds Google Cloud Storage configuration
type StorageConfig struct {
	BucketName      string
	ProjectID       string
	CredentialsJSON string // Optional: path to service account JSON or inline JSON
	UseDefaultCreds bool   // Use default application credentials
}

// StorageClient handles document storage operations in Google Cloud Storage
type StorageClient struct {
	client     *storage.Client
	bucketName string
	ctx        context.Context
}

// NewStorageClient creates a new Google Cloud Storage client
func NewStorageClient(config StorageConfig) (*StorageClient, error) {
	ctx := context.Background()

	var client *storage.Client
	var err error

	if config.UseDefaultCreds {
		// Use default application credentials (from GOOGLE_APPLICATION_CREDENTIALS env var)
		client, err = storage.NewClient(ctx)
	} else if config.CredentialsJSON != "" {
		// Use provided credentials JSON
		client, err = storage.NewClient(ctx, option.WithCredentialsJSON([]byte(config.CredentialsJSON)))
	} else {
		return nil, fmt.Errorf("no credentials provided for Google Cloud Storage")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	// Test bucket access
	bucket := client.Bucket(config.BucketName)
	if _, err := bucket.Attrs(ctx); err != nil {
		logrus.Warnf("Cannot access bucket %s: %v. Bucket may not exist or credentials may be invalid", config.BucketName, err)
	}

	logrus.Infof("Google Cloud Storage client initialized for bucket: %s", config.BucketName)

	return &StorageClient{
		client:     client,
		bucketName: config.BucketName,
		ctx:        ctx,
	}, nil
}

// UploadDocument uploads a document to Google Cloud Storage
// Returns the GCS path (object name) and file size on success
func (s *StorageClient) UploadDocument(agentID uuid.UUID, fileName string, fileData io.Reader, contentType string) (string, int64, error) {
	// Generate unique object name: documents/{agentID}/{timestamp}_{uuid}_{filename}
	timestamp := time.Now().Unix()
	fileUUID := uuid.New().String()
	objectName := fmt.Sprintf("documents/%s/%d_%s_%s", agentID.String(), timestamp, fileUUID, fileName)

	logrus.Infof("Uploading document to GCS: %s/%s", s.bucketName, objectName)

	// Create object writer
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(objectName)
	writer := obj.NewWriter(s.ctx)

	// Set content type
	writer.ObjectAttrs.ContentType = contentType
	writer.ObjectAttrs.Metadata = map[string]string{
		"agent_id":    agentID.String(),
		"uploaded_at": time.Now().Format(time.RFC3339),
	}

	// Copy file data to GCS and count bytes
	bytesWritten, err := io.Copy(writer, fileData)
	if err != nil {
		writer.Close()
		return "", 0, fmt.Errorf("failed to write to GCS: %w", err)
	}

	logrus.Infof("Copied %d bytes to GCS writer for %s", bytesWritten, objectName)

	// Close the writer to finalize upload
	if err := writer.Close(); err != nil {
		return "", 0, fmt.Errorf("failed to close GCS writer: %w", err)
	}

	// Get object attributes to retrieve file size
	attrs, err := s.client.Bucket(s.bucketName).Object(objectName).Attrs(s.ctx)
	if err != nil {
		logrus.Warnf("Failed to get object attributes for %s: %v", objectName, err)
		// Use the bytes written as fallback
		logrus.Infof("Document uploaded successfully to: %s (size: %d bytes from io.Copy)", objectName, bytesWritten)
		return objectName, bytesWritten, nil
	}

	logrus.Infof("Document uploaded successfully to: %s (size: %d bytes from attrs, %d bytes written)", objectName, attrs.Size, bytesWritten)
	return objectName, attrs.Size, nil
}

// DownloadDocument downloads a document from Google Cloud Storage
func (s *StorageClient) DownloadDocument(objectName string) (io.ReadCloser, error) {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(objectName)

	reader, err := obj.NewReader(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader for %s: %w", objectName, err)
	}

	return reader, nil
}

// DeleteDocument deletes a document from Google Cloud Storage
func (s *StorageClient) DeleteDocument(objectName string) error {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(objectName)

	if err := obj.Delete(s.ctx); err != nil {
		return fmt.Errorf("failed to delete %s: %w", objectName, err)
	}

	logrus.Infof("Document deleted from GCS: %s", objectName)
	return nil
}

// GetSignedURL generates a signed URL for temporary access to a document
func (s *StorageClient) GetSignedURL(objectName string, expiration time.Duration) (string, error) {
	// Generate signed URL using package-level function
	url, err := storage.SignedURL(s.bucketName, objectName, &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiration),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return url, nil
}

// ListDocuments lists all documents for a specific agent
func (s *StorageClient) ListDocuments(agentID uuid.UUID) ([]string, error) {
	prefix := fmt.Sprintf("documents/%s/", agentID.String())

	bucket := s.client.Bucket(s.bucketName)
	query := &storage.Query{Prefix: prefix}

	it := bucket.Objects(s.ctx, query)

	var documents []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list documents: %w", err)
		}
		documents = append(documents, attrs.Name)
	}

	return documents, nil
}

// GetDocumentMetadata retrieves metadata for a document
func (s *StorageClient) GetDocumentMetadata(objectName string) (map[string]string, error) {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(objectName)

	attrs, err := obj.Attrs(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get document metadata: %w", err)
	}

	metadata := make(map[string]string)
	metadata["content_type"] = attrs.ContentType
	metadata["size"] = fmt.Sprintf("%d", attrs.Size)
	metadata["created"] = attrs.Created.Format(time.RFC3339)
	metadata["updated"] = attrs.Updated.Format(time.RFC3339)

	// Add custom metadata
	for k, v := range attrs.Metadata {
		metadata[k] = v
	}

	return metadata, nil
}

// Close closes the storage client
func (s *StorageClient) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
