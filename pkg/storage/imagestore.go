package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gote/pkg/crypto"
	"gote/pkg/models"
	"gote/pkg/utils"
)

// ImageStore manages encrypted image storage
type ImageStore struct {
	dataDir string
	mutex   sync.RWMutex
	key     []byte
}

// EncryptedImage represents an encrypted image for storage
type EncryptedImage struct {
	ID            string    `json:"id"`
	Filename      string    `json:"filename"`
	ContentType   string    `json:"content_type"`
	Size          int64     `json:"size"`
	EncryptedData string    `json:"encrypted_data"`
	CreatedAt     time.Time `json:"created_at"`
}

// NewImageStore creates a new image store instance
func NewImageStore(dataDir string) *ImageStore {
	imageDir := filepath.Join(dataDir, "images")

	// Create images directory if it doesn't exist
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		// Log the error but don't panic - return the store anyway
		// The error will be caught when actually trying to save images
		fmt.Printf("Warning: Failed to create images directory %s: %v\n", imageDir, err)
	}

	return &ImageStore{
		dataDir: imageDir,
	}
}

// SetKey sets the encryption key for the image store
func (is *ImageStore) SetKey(key []byte) {
	is.mutex.Lock()
	defer is.mutex.Unlock()
	is.key = key
}

// StoreImage encrypts and stores an image, returning the image metadata
func (is *ImageStore) StoreImage(imageData []byte, contentType, filename string) (*models.Image, error) {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	if is.key == nil {
		return nil, fmt.Errorf("encryption key not set")
	}

	// Generate unique ID for the image
	imageID := utils.GenerateShortUUID()

	// Create image metadata
	image := &models.Image{
		ID:          imageID,
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(imageData)),
		CreatedAt:   time.Now(),
	}

	// Encrypt image data directly (no base64 encoding before encryption)
	encryptedData, err := crypto.EncryptBytes(imageData, is.key)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt image: %v", err)
	}

	// Create encrypted image struct
	encryptedImage := &EncryptedImage{
		ID:            image.ID,
		Filename:      image.Filename,
		ContentType:   image.ContentType,
		Size:          image.Size,
		EncryptedData: encryptedData,
		CreatedAt:     image.CreatedAt,
	}

	// Save encrypted image to disk
	imagePath := filepath.Join(is.dataDir, fmt.Sprintf("%s.json", imageID))

	// Ensure directory exists before saving
	if err := os.MkdirAll(is.dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create images directory: %v", err)
	}

	if err := is.saveEncryptedImageToDisk(imagePath, encryptedImage); err != nil {
		return nil, fmt.Errorf("failed to save image: %v", err)
	}

	return image, nil
}

// GetImage retrieves and decrypts an image by ID
func (is *ImageStore) GetImage(imageID string) ([]byte, *models.Image, error) {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	if is.key == nil {
		return nil, nil, fmt.Errorf("encryption key not set")
	}

	imagePath := filepath.Join(is.dataDir, fmt.Sprintf("%s.json", imageID))

	// Load encrypted image from disk
	encryptedImage, err := is.loadEncryptedImageFromDisk(imagePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load image: %v", err)
	}

	// Decrypt image data directly to bytes
	imageData, err := crypto.DecryptBytes(encryptedImage.EncryptedData, is.key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt image: %v", err)
	}

	// Create image metadata
	image := &models.Image{
		ID:          encryptedImage.ID,
		Filename:    encryptedImage.Filename,
		ContentType: encryptedImage.ContentType,
		Size:        encryptedImage.Size,
		CreatedAt:   encryptedImage.CreatedAt,
	}

	return imageData, image, nil
}

// DeleteImage removes an image from storage
func (is *ImageStore) DeleteImage(imageID string) error {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	imagePath := filepath.Join(is.dataDir, fmt.Sprintf("%s.json", imageID))
	return os.Remove(imagePath)
}

// ListImages returns a list of all stored images (metadata only)
func (is *ImageStore) ListImages() ([]*models.Image, error) {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	files, err := os.ReadDir(is.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read images directory: %v", err)
	}

	var images []*models.Image
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		imagePath := filepath.Join(is.dataDir, file.Name())
		encryptedImage, err := is.loadEncryptedImageFromDisk(imagePath)
		if err != nil {
			continue // Skip corrupted files
		}

		image := &models.Image{
			ID:          encryptedImage.ID,
			Filename:    encryptedImage.Filename,
			ContentType: encryptedImage.ContentType,
			Size:        encryptedImage.Size,
			CreatedAt:   encryptedImage.CreatedAt,
		}
		images = append(images, image)
	}

	return images, nil
}

// saveEncryptedImageToDisk saves an encrypted image to disk
func (is *ImageStore) saveEncryptedImageToDisk(path string, encryptedImage *EncryptedImage) error {
	data, err := json.MarshalIndent(encryptedImage, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// loadEncryptedImageFromDisk loads an encrypted image from disk
func (is *ImageStore) loadEncryptedImageFromDisk(path string) (*EncryptedImage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var encryptedImage EncryptedImage
	if err := json.Unmarshal(data, &encryptedImage); err != nil {
		return nil, err
	}

	return &encryptedImage, nil
}
