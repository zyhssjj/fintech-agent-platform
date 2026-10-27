import { useApiResource } from "@/base/composables/useApiResource";
import { meetingApi } from "@/module-meeting/api";

// 本文件提供会议详情的 compose/hook。
export function useMeetingDetail(meetingId: string) {
  return useApiResource(() => meetingApi.getMeeting(meetingId), [meetingId]);
}
