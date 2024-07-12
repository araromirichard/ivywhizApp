package uploader

import (
	"context"
	"io"
	"log"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type ImageUploaderService struct {
	cloud *cloudinary.Cloudinary
}

func New(cloudName, apiKey, apiSecret string) *ImageUploaderService {
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudinary, %v", err)
	}

	return &ImageUploaderService{cloud: cld}
}

// UploadImage uploads an image to Cloudinary and returns the secure URL and public ID
func (i *ImageUploaderService) UploadImage(ctx context.Context, file io.Reader) (string, string, error) {
	uploadParams := uploader.UploadParams{}
	uploadResult, err := i.cloud.Upload.Upload(ctx, file, uploadParams)
	if err != nil {
		log.Printf("Failed to upload image to Cloudinary: %v", err)
		return "", "", err
	}

	return uploadResult.SecureURL, uploadResult.PublicID, nil
}

// DeleteImage deletes an image from Cloudinary using the public ID
func (i *ImageUploaderService) DeleteImage(ctx context.Context, publicID string) error {
	_, err := i.cloud.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: publicID})
	if err != nil {
		log.Printf("Failed to delete image from Cloudinary: %v", err)
		return err
	}
	return nil
}

// UploadOrUpdateImage uploads an image to Cloudinary, deleting the existing one if it exists
func (i *ImageUploaderService) UploadOrUpdateImage(ctx context.Context, file io.Reader, existingPublicID string) (string, string, error) {
	if existingPublicID != "" {
		// Check if the photo exists and delete it
		err := i.DeleteImage(ctx, existingPublicID)
		if err != nil {
			log.Printf("Failed to delete existing image: %v", err)
			// Continue to upload the new image even if deletion fails
		}
	}

	// Upload the new image
	secureURL, publicID, err := i.UploadImage(ctx, file)
	if err != nil {
		return "", "", err
	}

	return secureURL, publicID, nil
}
