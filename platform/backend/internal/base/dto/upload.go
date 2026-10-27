package dto

// 本文件定义通用分片上传 UPLOAD-001~UPLOAD-003 的传输结构。
type InitUploadRequest struct {
	FileName   string `json:"file_name" binding:"required"`
	FileSize   int64  `json:"file_size" binding:"required"`
	ChunkCount int    `json:"chunk_count" binding:"required"`
}

type InitUploadResponse struct {
	UploadID  string `json:"upload_id"`
	ChunkSize int64  `json:"chunk_size"`
	ExpiresAt string `json:"expires_at"`
	UploadURL string `json:"upload_url"`
}

type ChunkUploadResponse struct {
	UploadID   string `json:"upload_id"`
	ChunkIndex int    `json:"chunk_index"`
	Received   bool   `json:"received"`
}

type CompleteUploadResponse struct {
	FileID   string `json:"file_id"`
	FileURL  string `json:"file_url"`
	FileSize int64  `json:"file_size"`
	MD5      string `json:"md5"`
	Status   string `json:"status"`
}
