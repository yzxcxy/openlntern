"use client";

import { useCallback, useRef, useState, type ChangeEvent } from "react";
import { requestBackend, type RouterLike } from "../auth";
import {
  createMessageId,
  inferUploadKind,
  normalizeMimeType,
  type BackendChatUploadAsset,
  type UploadAssetItem,
} from "./chat-helpers";

// 附件上传状态与流程收口到 hook，页面只组合发送逻辑和失败回滚。

export function usePendingUploads({
  threadId,
  router,
  userId,
  uploadsBlocked,
}: {
  threadId: string;
  router: RouterLike;
  userId: string;
  uploadsBlocked: boolean;
}) {
  const uploadInputRef = useRef<HTMLInputElement | null>(null);
  const uploadEpochRef = useRef(0);
  const [pendingUploads, setPendingUploads] = useState<UploadAssetItem[]>([]);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");

  const clearUploadError = useCallback(() => {
    setUploadError("");
  }, []);

  const clearPendingUploads = useCallback(() => {
    uploadEpochRef.current += 1;
    setPendingUploads([]);
  }, []);

  const resetUploads = useCallback(() => {
    uploadEpochRef.current += 1;
    setPendingUploads([]);
    setUploadError("");
    setUploading(false);
  }, []);

  const restorePendingUploads = useCallback((items: UploadAssetItem[]) => {
    setPendingUploads(items);
  }, []);

  const handleOpenUploadPicker = useCallback(() => {
    if (uploadsBlocked || uploading) {
      return;
    }
    uploadInputRef.current?.click();
  }, [uploading, uploadsBlocked]);

  const removePendingUpload = useCallback((assetId: string) => {
    setPendingUploads((current) => current.filter((item) => item.id !== assetId));
  }, []);

  // 上传单个附件并返回规范化后的上传结果。
  const uploadSingleAsset = useCallback(
    async (file: File): Promise<UploadAssetItem> => {
      const formData = new FormData();
      formData.append("file", file);
      if (threadId) {
        formData.append("thread_id", threadId);
      }
      const result = await requestBackend<BackendChatUploadAsset>("/v1/chat/uploads", {
        method: "POST",
        body: formData,
        fallbackMessage: `上传失败：${file.name}`,
        router,
        userId,
      });
      if (!result.data?.url) {
        const backendMessage =
          result?.message && typeof result.message === "string"
            ? result.message
            : "";
        throw new Error(backendMessage || `上传失败：${file.name}`);
      }

      const normalizedMimeType = normalizeMimeType(
        result.data.mime_type || file.type || "application/octet-stream"
      );
      const normalizedFileName =
        (typeof result.data.file_name === "string" && result.data.file_name.trim()) ||
        file.name ||
        "attachment";
      const normalizedSize =
        typeof result.data.size === "number" && Number.isFinite(result.data.size)
          ? result.data.size
          : file.size;
      return {
        id: createMessageId(),
        key:
          (typeof result.data.key === "string" && result.data.key.trim()) ||
          normalizedFileName,
        url: String(result.data.url),
        mimeType: normalizedMimeType,
        fileName: normalizedFileName,
        size: normalizedSize,
        mediaKind:
          result.data.media_kind ||
          inferUploadKind(normalizedMimeType || "application/octet-stream"),
      };
    },
    [router, threadId, userId]
  );

  // 处理文件选择并批量上传，成功项加入待发送附件区。
  const handleUploadInputChange = useCallback(
    async (event: ChangeEvent<HTMLInputElement>) => {
      const selectedFiles = Array.from(event.target.files ?? []);
      event.target.value = "";
      if (selectedFiles.length === 0) {
        return;
      }

      const uploadEpoch = uploadEpochRef.current;
      setUploadError("");
      setUploading(true);
      const successItems: UploadAssetItem[] = [];
      let firstError = "";
      for (const file of selectedFiles) {
        try {
          const uploaded = await uploadSingleAsset(file);
          successItems.push(uploaded);
        } catch (error) {
          if (!firstError) {
            firstError = error instanceof Error ? error.message : `上传失败：${file.name}`;
          }
        }
      }
      if (uploadEpochRef.current !== uploadEpoch) {
        return;
      }
      setUploading(false);

      if (successItems.length > 0) {
        setPendingUploads((current) => {
          const merged = [...current];
          successItems.forEach((item) => {
            const existed = merged.some((existing) => existing.url === item.url);
            if (!existed) {
              merged.push(item);
            }
          });
          return merged;
        });
      }
      if (firstError) {
        setUploadError(firstError);
      }
    },
    [uploadSingleAsset]
  );

  return {
    uploadInputRef,
    pendingUploads,
    uploading,
    uploadError,
    handleOpenUploadPicker,
    handleUploadInputChange,
    removePendingUpload,
    clearUploadError,
    clearPendingUploads,
    resetUploads,
    restorePendingUploads,
  };
}
