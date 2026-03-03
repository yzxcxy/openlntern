"use client";

export const THREAD_HISTORY_UPSERT_EVENT = "thread-history-upsert";

export type ThreadHistoryItem = {
  thread_id?: string;
  title?: string;
  updated_at?: string;
  created_at?: string;
  pending_title?: boolean;
};

export const dispatchThreadHistoryUpsert = (detail: ThreadHistoryItem) => {
  if (typeof window === "undefined" || !detail.thread_id) {
    return;
  }
  window.dispatchEvent(
    new CustomEvent<ThreadHistoryItem>(THREAD_HISTORY_UPSERT_EVENT, {
      detail,
    })
  );
};
