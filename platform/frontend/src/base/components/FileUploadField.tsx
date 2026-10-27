import React from "react";
import { FileUp, X } from "lucide-react";
import { uploadApi } from "@/base/api";
import type { CompleteUploadResponse } from "@/base/api";
import { useUIStore } from "@/base/stores/uiStore";

export interface FileUploadFieldProps {
  accept?: string;
  disabled?: boolean;
  maxSizeMB?: number;
  onUploaded?: (file: CompleteUploadResponse) => void;
  onSelect?: (files: FileList) => void;
}

export function FileUploadField({ accept, disabled, maxSizeMB = 50, onUploaded, onSelect }: FileUploadFieldProps) {
  const [file, setFile] = React.useState<File | null>(null);
  const [uploading, setUploading] = React.useState(false);
  const notify = useUIStore((state) => state.notify);

  async function uploadSelected(nextFile: File) {
    if (nextFile.size > maxSizeMB * 1024 * 1024) {
      notify({ type: "warning", title: "文件过大", message: `请选择不超过 ${maxSizeMB}MB 的文件` });
      return;
    }

    setUploading(true);
    try {
      const init = await uploadApi.init({ file_name: nextFile.name, file_size: nextFile.size, chunk_count: 1 });
      await uploadApi.chunk(init.upload_id, 0, nextFile);
      const completed = await uploadApi.complete(init.upload_id);
      notify({ type: "success", title: "上传完成", message: nextFile.name });
      onUploaded?.(completed);
    } finally {
      setUploading(false);
    }
  }

  return (
    <div className="file-upload-shell rounded-[8px] border border-dashed border-slate-300 bg-slate-50 p-4">
      <label className="file-upload-label flex cursor-pointer flex-col items-center justify-center gap-2 text-center">
        <FileUp className="h-7 w-7 text-slate-500" />
        <span className="text-sm font-semibold text-slate-700">{uploading ? "上传中..." : "选择文件上传"}</span>
        <span className="text-xs text-slate-500">最大 {maxSizeMB}MB，选择后自动调用基座上传 API</span>
        <input
          className="sr-only"
          type="file"
          accept={accept}
          disabled={disabled || uploading}
          onChange={(event) => {
            const files = event.currentTarget.files;
            if (!files?.length) return;
            setFile(files[0]);
            onSelect?.(files);
            void uploadSelected(files[0]);
          }}
        />
      </label>
      {file && (
        <div className="mt-3 flex items-center justify-between rounded-[6px] bg-white px-3 py-2 text-sm text-slate-700">
          <span className="truncate">{file.name}</span>
          <button className="rounded p-1 hover:bg-slate-100" onClick={() => setFile(null)} type="button" title="移除文件">
            <X className="h-4 w-4" />
          </button>
        </div>
      )}
    </div>
  );
}
