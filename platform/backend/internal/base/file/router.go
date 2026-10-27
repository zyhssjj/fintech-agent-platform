package file

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wfagent/internal/base/config"
	"wfagent/internal/base/dto"
	"wfagent/internal/base/response"
)

// RegisterRoutes registers common upload endpoints.
func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	handler := NewHandler(NewService(db))
	group := v1.Group("/upload")
	group.POST("/init", handler.InitUpload)
	group.POST("/:upload_id/chunk", handler.UploadChunk)
	group.POST("/:upload_id/complete", handler.CompleteUpload)

	files := v1.Group("/files")
	files.POST("", handler.Upload)
	files.GET("/preview/*object_name", handler.Preview)
	files.GET("/download/*object_name", handler.Download)
	files.DELETE("/*object_name", handler.Delete)
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type Service struct {
	db     *gorm.DB
	client *MinIOClient
}

type uploadMeta struct {
	UploadID   string    `json:"upload_id"`
	FileName   string    `json:"file_name"`
	FileSize   int64     `json:"file_size"`
	ChunkCount int       `json:"chunk_count"`
	CreatedAt  time.Time `json:"created_at"`
}

type fileRecord struct {
	FileID      string    `json:"file_id"`
	FileName    string    `json:"file_name"`
	ObjectName  string    `json:"object_name"`
	FileURL     string    `json:"file_url"`
	FileSize    int64     `json:"file_size"`
	MD5         string    `json:"md5"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

var indexMu sync.Mutex

func NewService(db *gorm.DB) *Service {
	var client *MinIOClient
	if config.C.File.StorageDriver == "minio" {
		if minioClient, err := NewMinIOClient(config.C.File); err == nil {
			client = minioClient
		}
	}
	return &Service{db: db, client: client}
}

// InitUpload godoc
// @Summary Initialize chunked upload
// @Tags Upload
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.InitUploadRequest true "Upload metadata"
// @Success 200 {object} response.Body{data=dto.InitUploadResponse}
// @Failure 400 {object} response.Body
// @Failure 401 {object} response.Body
// @Router /v1/upload/init [post]
func (h *Handler) InitUpload(c *gin.Context) {
	var req dto.InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	data, err := h.service.InitUpload(req)
	if err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.Success(c, data)
}

// UploadChunk godoc
// @Summary Upload file chunk
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param upload_id path string true "Upload ID"
// @Param chunk_index formData int true "Chunk index"
// @Param file formData file true "File chunk"
// @Success 200 {object} response.Body{data=dto.ChunkUploadResponse}
// @Failure 400 {object} response.Body
// @Failure 401 {object} response.Body
// @Router /v1/upload/{upload_id}/chunk [post]
func (h *Handler) UploadChunk(c *gin.Context) {
	data, err := h.service.UploadChunk(c)
	if err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.Success(c, data)
}

// CompleteUpload godoc
// @Summary Complete chunked upload
// @Tags Upload
// @Produce json
// @Security BearerAuth
// @Param upload_id path string true "Upload ID"
// @Success 200 {object} response.Body{data=dto.CompleteUploadResponse}
// @Failure 401 {object} response.Body
// @Failure 500 {object} response.Body
// @Router /v1/upload/{upload_id}/complete [post]
func (h *Handler) CompleteUpload(c *gin.Context) {
	data, err := h.service.CompleteUpload(c, c.Param("upload_id"))
	if err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.Success(c, data)
}

// Upload godoc
// @Summary Upload a file to MinIO
// @Tags Files
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "File to upload"
// @Success 201 {object} response.Body{data=dto.CompleteUploadResponse}
// @Failure 400 {object} response.Body
// @Failure 401 {object} response.Body
// @Router /v1/files [post]
func (h *Handler) Upload(c *gin.Context) {
	data, err := h.service.Upload(c)
	if err != nil {
		response.Error(c, response.ErrValidation, err.Error())
		return
	}
	response.Created(c, data)
}

// Preview godoc
// @Summary Create a temporary file preview URL
// @Tags Files
// @Produce json
// @Security BearerAuth
// @Param object_name path string true "Object name"
// @Success 200 {object} response.Body{data=map[string]interface{}}
// @Failure 401 {object} response.Body
// @Failure 404 {object} response.Body
// @Router /v1/files/preview/{object_name} [get]
func (h *Handler) Preview(c *gin.Context) {
	h.presign(c)
}

// Download godoc
// @Summary Create a temporary file download URL
// @Tags Files
// @Produce json
// @Security BearerAuth
// @Param object_name path string true "Object name"
// @Success 200 {object} response.Body{data=map[string]interface{}}
// @Failure 401 {object} response.Body
// @Failure 404 {object} response.Body
// @Router /v1/files/download/{object_name} [get]
func (h *Handler) Download(c *gin.Context) {
	h.presign(c)
}

// Delete godoc
// @Summary Delete a file from MinIO
// @Tags Files
// @Produce json
// @Security BearerAuth
// @Param object_name path string true "Object name"
// @Success 204
// @Failure 401 {object} response.Body
// @Failure 404 {object} response.Body
// @Router /v1/files/{object_name} [delete]
func (h *Handler) Delete(c *gin.Context) {
	objectName := strings.TrimPrefix(c.Param("object_name"), "/")
	if err := h.service.Delete(c, objectName); err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.NoContent(c)
}

func (h *Handler) presign(c *gin.Context) {
	objectName := strings.TrimPrefix(c.Param("object_name"), "/")
	url, err := h.service.PresignedURL(c, objectName, 15*time.Minute)
	if err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.Success(c, gin.H{"url": url, "expires_in": 900})
}

func (s *Service) InitUpload(req dto.InitUploadRequest) (dto.InitUploadResponse, error) {
	if req.FileSize <= 0 {
		return dto.InitUploadResponse{}, fmt.Errorf("file_size must be greater than zero")
	}
	if req.ChunkCount <= 0 {
		return dto.InitUploadResponse{}, fmt.Errorf("chunk_count must be greater than zero")
	}
	if err := s.validateFileName(req.FileName); err != nil {
		return dto.InitUploadResponse{}, err
	}
	uploadID := "upload_" + randomID(12)
	meta := uploadMeta{
		UploadID:   uploadID,
		FileName:   filepath.Base(req.FileName),
		FileSize:   req.FileSize,
		ChunkCount: req.ChunkCount,
		CreatedAt:  time.Now(),
	}
	if err := os.MkdirAll(s.uploadDir(uploadID), 0o755); err != nil {
		return dto.InitUploadResponse{}, err
	}
	if err := s.writeMeta(meta); err != nil {
		return dto.InitUploadResponse{}, err
	}
	return dto.InitUploadResponse{
		UploadID:  uploadID,
		ChunkSize: 5 * 1024 * 1024,
		ExpiresAt: time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		UploadURL: "/v1/upload/" + uploadID + "/chunk",
	}, nil
}

func (s *Service) UploadChunk(c *gin.Context) (dto.ChunkUploadResponse, error) {
	uploadID := c.Param("upload_id")
	if uploadID == "" {
		return dto.ChunkUploadResponse{}, fmt.Errorf("upload_id is required")
	}
	if _, err := s.readMeta(uploadID); err != nil {
		return dto.ChunkUploadResponse{}, err
	}
	chunkIndex, err := strconv.Atoi(c.DefaultPostForm("chunk_index", c.Query("chunk_index")))
	if err != nil {
		return dto.ChunkUploadResponse{}, fmt.Errorf("chunk_index is required")
	}
	if chunkIndex < 0 {
		return dto.ChunkUploadResponse{}, fmt.Errorf("chunk_index must not be negative")
	}
	dstPath := s.chunkPath(uploadID, chunkIndex)
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return dto.ChunkUploadResponse{}, err
	}

	src, err := chunkReader(c)
	if err != nil {
		return dto.ChunkUploadResponse{}, err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return dto.ChunkUploadResponse{}, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return dto.ChunkUploadResponse{}, err
	}
	return dto.ChunkUploadResponse{UploadID: uploadID, ChunkIndex: chunkIndex, Received: true}, nil
}

func (s *Service) CompleteUpload(c *gin.Context, uploadID string) (dto.CompleteUploadResponse, error) {
	meta, err := s.readMeta(uploadID)
	if err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	if config.C.File.StorageDriver != "minio" {
		return dto.CompleteUploadResponse{}, fmt.Errorf("file storage driver must be minio")
	}
	if s.client == nil {
		return dto.CompleteUploadResponse{}, fmt.Errorf("minio client is not initialized")
	}

	finalPath := filepath.Join(s.uploadDir(uploadID), "merged")
	final, err := os.Create(finalPath)
	if err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	hash := md5.New()
	writer := io.MultiWriter(final, hash)
	for i := 0; i < meta.ChunkCount; i++ {
		chunk, err := os.Open(s.chunkPath(uploadID, i))
		if err != nil {
			final.Close()
			return dto.CompleteUploadResponse{}, fmt.Errorf("missing chunk %d: %w", i, err)
		}
		if _, err := io.Copy(writer, chunk); err != nil {
			chunk.Close()
			final.Close()
			return dto.CompleteUploadResponse{}, err
		}
		chunk.Close()
	}
	if err := final.Close(); err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	stat, err := os.Stat(finalPath)
	if err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	if stat.Size() != meta.FileSize {
		return dto.CompleteUploadResponse{}, fmt.Errorf("merged file size mismatch: expected %d got %d", meta.FileSize, stat.Size())
	}
	md5Value := hex.EncodeToString(hash.Sum(nil))
	if record, ok := s.findByMD5(md5Value); ok {
		_ = os.RemoveAll(s.uploadDir(uploadID))
		return dto.CompleteUploadResponse{
			FileID:   record.FileID,
			FileURL:  record.FileURL,
			FileSize: record.FileSize,
			MD5:      record.MD5,
			Status:   "deduplicated",
		}, nil
	}

	file, err := os.Open(finalPath)
	if err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	defer file.Close()

	fileID := "file_" + randomID(16)
	objectName := fmt.Sprintf("uploads/%s/%s-%s", time.Now().Format("2006/01/02"), fileID, meta.FileName)
	contentType := mime.TypeByExtension(filepath.Ext(meta.FileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if _, err := s.client.Upload(c.Request.Context(), objectName, file, stat.Size(), contentType); err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	fileURL, err := s.client.GetPresignedURL(c.Request.Context(), objectName, 24*time.Hour)
	if err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	_ = s.saveRecord(fileRecord{
		FileID:      fileID,
		FileName:    meta.FileName,
		ObjectName:  objectName,
		FileURL:     fileURL,
		FileSize:    stat.Size(),
		MD5:         md5Value,
		ContentType: contentType,
		CreatedAt:   time.Now(),
	})
	_ = os.RemoveAll(s.uploadDir(uploadID))
	return dto.CompleteUploadResponse{
		FileID:   fileID,
		FileURL:  fileURL,
		FileSize: stat.Size(),
		MD5:      md5Value,
		Status:   "completed",
	}, nil
}

func (s *Service) Upload(c *gin.Context) (dto.CompleteUploadResponse, error) {
	if config.C.File.StorageDriver != "minio" {
		return dto.CompleteUploadResponse{}, fmt.Errorf("file storage driver must be minio")
	}
	if s.client == nil {
		return dto.CompleteUploadResponse{}, fmt.Errorf("minio client is not initialized")
	}
	header, err := c.FormFile("file")
	if err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	if header.Size > config.C.File.MaxUploadMB*1024*1024 {
		return dto.CompleteUploadResponse{}, fmt.Errorf("file exceeds max upload size")
	}
	if err := s.validateFileName(header.Filename); err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	src, err := header.Open()
	if err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	defer src.Close()

	tmpDir := filepath.Join(config.C.File.UploadDir, "single")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	tmpPath := filepath.Join(tmpDir, "upload_"+randomID(12))
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	hash := md5.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hash), src); err != nil {
		tmp.Close()
		return dto.CompleteUploadResponse{}, err
	}
	if err := tmp.Close(); err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	defer os.Remove(tmpPath)

	md5Value := hex.EncodeToString(hash.Sum(nil))
	if record, ok := s.findByMD5(md5Value); ok {
		return dto.CompleteUploadResponse{FileID: record.FileID, FileURL: record.FileURL, FileSize: record.FileSize, MD5: record.MD5, Status: "deduplicated"}, nil
	}
	file, err := os.Open(tmpPath)
	if err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(header.Filename))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	fileID := "file_" + randomID(16)
	objectName := fmt.Sprintf("uploads/%s/%s-%s", time.Now().Format("2006/01/02"), fileID, filepath.Base(header.Filename))
	if _, err := s.client.Upload(c.Request.Context(), objectName, file, header.Size, contentType); err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	fileURL, err := s.client.GetPresignedURL(c.Request.Context(), objectName, 24*time.Hour)
	if err != nil {
		return dto.CompleteUploadResponse{}, err
	}
	_ = s.saveRecord(fileRecord{FileID: fileID, FileName: filepath.Base(header.Filename), ObjectName: objectName, FileURL: fileURL, FileSize: header.Size, MD5: md5Value, ContentType: contentType, CreatedAt: time.Now()})
	return dto.CompleteUploadResponse{FileID: fileID, FileURL: fileURL, FileSize: header.Size, MD5: md5Value, Status: "completed"}, nil
}

func (s *Service) PresignedURL(c *gin.Context, objectName string, expiry time.Duration) (string, error) {
	if objectName == "" {
		return "", fmt.Errorf("object_name is required")
	}
	if s.client == nil {
		return "", fmt.Errorf("minio client is not initialized")
	}
	return s.client.GetPresignedURL(c.Request.Context(), objectName, expiry)
}

func (s *Service) Delete(c *gin.Context, objectName string) error {
	if objectName == "" {
		return fmt.Errorf("object_name is required")
	}
	if s.client == nil {
		return fmt.Errorf("minio client is not initialized")
	}
	if err := s.client.Delete(c.Request.Context(), objectName); err != nil {
		return err
	}
	return s.deleteRecord(objectName)
}

func (s *Service) validateFileName(fileName string) error {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		return fmt.Errorf("file extension is required")
	}
	for _, allowed := range config.C.File.AllowedTypes {
		if strings.ToLower(strings.TrimSpace(allowed)) == ext {
			return nil
		}
	}
	return fmt.Errorf("file type %s is not allowed", ext)
}

func (s *Service) uploadDir(uploadID string) string {
	return filepath.Join(config.C.File.UploadDir, "chunks", uploadID)
}

func (s *Service) chunkPath(uploadID string, chunkIndex int) string {
	return filepath.Join(s.uploadDir(uploadID), fmt.Sprintf("%06d.part", chunkIndex))
}

func (s *Service) metaPath(uploadID string) string {
	return filepath.Join(s.uploadDir(uploadID), "meta.json")
}

func (s *Service) indexPath() string {
	return filepath.Join(config.C.File.UploadDir, "files_index.json")
}

func (s *Service) writeMeta(meta uploadMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(meta.UploadID), data, 0o644)
}

func (s *Service) readMeta(uploadID string) (uploadMeta, error) {
	data, err := os.ReadFile(s.metaPath(uploadID))
	if err != nil {
		return uploadMeta{}, err
	}
	var meta uploadMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return uploadMeta{}, err
	}
	return meta, nil
}

func (s *Service) findByMD5(value string) (fileRecord, bool) {
	indexMu.Lock()
	defer indexMu.Unlock()
	records := s.readIndexLocked()
	for _, record := range records {
		if record.MD5 == value {
			return record, true
		}
	}
	return fileRecord{}, false
}

func (s *Service) saveRecord(record fileRecord) error {
	indexMu.Lock()
	defer indexMu.Unlock()
	records := s.readIndexLocked()
	records = append(records, record)
	return s.writeIndexLocked(records)
}

func (s *Service) deleteRecord(objectName string) error {
	indexMu.Lock()
	defer indexMu.Unlock()
	records := s.readIndexLocked()
	next := records[:0]
	for _, record := range records {
		if record.ObjectName != objectName {
			next = append(next, record)
		}
	}
	return s.writeIndexLocked(next)
}

func (s *Service) readIndexLocked() []fileRecord {
	data, err := os.ReadFile(s.indexPath())
	if err != nil {
		return []fileRecord{}
	}
	var records []fileRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return []fileRecord{}
	}
	return records
}

func (s *Service) writeIndexLocked(records []fileRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.indexPath()), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.indexPath(), data, 0o644)
}

func chunkReader(c *gin.Context) (io.ReadCloser, error) {
	file, err := c.FormFile("file")
	if err == nil {
		return file.Open()
	}
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return nil, fmt.Errorf("file chunk is required")
	}
	return http.MaxBytesReader(c.Writer, c.Request.Body, config.C.File.MaxUploadMB*1024*1024), nil
}

func randomID(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
