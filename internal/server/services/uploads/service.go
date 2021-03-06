package uploads

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"salsa.debian.org/autodeb-team/autodeb/internal/filesystem"
	"salsa.debian.org/autodeb-team/autodeb/internal/server/database"
	"salsa.debian.org/autodeb-team/autodeb/internal/server/models"
	"salsa.debian.org/autodeb-team/autodeb/internal/server/services/jobs"
	"salsa.debian.org/autodeb-team/autodeb/internal/server/services/pgp"
)

//Service manages uploads
type Service struct {
	db          *database.Database
	pgpService  *pgp.Service
	jobsService *jobs.Service
	fs          filesystem.FS
}

//New creates a new upload service
func New(db *database.Database, pgpService *pgp.Service, jobsService *jobs.Service, fs filesystem.FS) *Service {
	service := &Service{
		db:          db,
		pgpService:  pgpService,
		jobsService: jobsService,
		fs:          fs,
	}
	return service
}

// FS returns the services's filesystem
func (service *Service) FS() filesystem.FS {
	return service.fs
}

// UploadedFilesDirectory contains files that are not yet associated
// with a package upload.
func (service *Service) UploadedFilesDirectory() string {
	return "/files"
}

// UploadsDirectory contains completed uploads.
func (service *Service) UploadsDirectory() string {
	return "/uploads"
}

// GetAllUploads returns all uploads
func (service *Service) GetAllUploads() ([]*models.Upload, error) {
	return service.db.GetAllUploads()
}

// GetAllUploadsPageLimit returns all uploads with pagination
func (service *Service) GetAllUploadsPageLimit(page, limit int) ([]*models.Upload, error) {
	return service.db.GetAllUploadsPageLimit(page, limit)
}

// GetAllUploadsByUserID returns all uploads for a given user id
func (service *Service) GetAllUploadsByUserID(userID uint) ([]*models.Upload, error) {
	return service.db.GetAllUploadsByUserID(userID)
}

// GetAllUploadsByUserIDPageLimit returns all uploads for a given user id with pagination
func (service *Service) GetAllUploadsByUserIDPageLimit(userID uint, page, limit int) ([]*models.Upload, error) {
	return service.db.GetAllUploadsByUserIDPageLimit(userID, page, limit)
}

// GetUpload returns an upload by ID
func (service *Service) GetUpload(uploadID uint) (*models.Upload, error) {
	return service.db.GetUpload(uploadID)
}

//GetAllFileUploadsByUploadID returns all FileUploads associated to an upload
func (service *Service) GetAllFileUploadsByUploadID(uploadID uint) ([]*models.FileUpload, error) {
	fileUploads, err := service.db.GetAllFileUploadsByUploadID(uploadID)
	if err != nil {
		return nil, err
	}
	return fileUploads, nil
}

// GetUploadDSC returns the DSC of the upload with a matching id
func (service *Service) GetUploadDSC(uploadID uint) (io.ReadCloser, error) {
	fileUploads, err := service.GetAllFileUploadsByUploadID(uploadID)
	if err != nil {
		return nil, err
	}

	for _, fileUpload := range fileUploads {
		if strings.HasSuffix(fileUpload.Filename, ".dsc") {
			return service.GetUploadFile(uploadID, fileUpload.Filename)
		}
	}

	return nil, nil
}

// GetUploadChanges returns the .changes of the upload with a matching id
func (service *Service) GetUploadChanges(uploadID uint) (io.ReadCloser, error) {
	fileUploads, err := service.GetAllFileUploadsByUploadID(uploadID)
	if err != nil {
		return nil, err
	}

	for _, fileUpload := range fileUploads {
		if strings.HasSuffix(fileUpload.Filename, ".changes") {
			return service.GetUploadFile(uploadID, fileUpload.Filename)
		}
	}

	return nil, nil
}

// GetUploadFile returns the file associated with the upload id and filename
func (service *Service) GetUploadFile(uploadID uint, filename string) (io.ReadCloser, error) {
	file, err := service.fs.Open(
		filepath.Join(
			service.UploadsDirectory(),
			fmt.Sprint(uploadID),
			filename,
		),
	)
	if err != nil {
		return nil, err
	}
	return file, nil
}
