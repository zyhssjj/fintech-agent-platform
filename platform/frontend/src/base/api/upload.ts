import type {
  ChunkUploadResponse,
  CompleteUploadResponse,
  InitUploadRequest,
  InitUploadResponse,
} from "@/base/api/contracts";
import { request } from "@/base/api/request";

// 本文件封装通用分片上传接口 UPLOAD-001~UPLOAD-003。
export const uploadApi = {
  init(payload: InitUploadRequest) {
    return request.post<InitUploadResponse>("/upload/init", payload);
  },
  chunk(uploadId: string, chunkIndex: number, file: Blob) {
    const formData = new FormData();
    formData.set("chunk_index", String(chunkIndex));
    formData.set("file", file);
    return request.upload<ChunkUploadResponse>(`/upload/${uploadId}/chunk`, formData);
  },
  complete(uploadId: string) {
    return request.post<CompleteUploadResponse>(`/upload/${uploadId}/complete`);
  },
};
