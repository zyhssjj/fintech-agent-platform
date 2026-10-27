import type {
  CreateMeetingResponse,
  CreateMeetingRequest,
  MeetingActionItemsResponse,
  MeetingDetail,
  TranscribeMeetingRequest,
  TranscribeMeetingResponse,
  UploadMeetingAudioResponse,
} from "@/base/api";
import { request } from "@/base/api";

// 本文件封装会议管理 MEETING-* 接口。
export const meetingApi = {
  createMeeting(payload: CreateMeetingRequest) {
    return request.post<CreateMeetingResponse>("/meetings", payload);
  },
  uploadAudio(meetingId: string, audio: Blob, duration: number) {
    const formData = new FormData();
    formData.set("audio", audio);
    formData.set("duration", String(duration));
    return request.upload<UploadMeetingAudioResponse>(`/meetings/${meetingId}/upload`, formData);
  },
  transcribe(meetingId: string, payload: TranscribeMeetingRequest) {
    return request.post<TranscribeMeetingResponse>(`/meetings/${meetingId}/transcribe`, payload);
  },
  getMeeting(meetingId: string) {
    return request.get<MeetingDetail>(`/meetings/${meetingId}`);
  },
  deleteMeeting(meetingId: string) {
    return request.delete(`/meetings/${meetingId}`);
  },
  listActionItems(meetingId: string) {
    return request.get<MeetingActionItemsResponse>(`/meetings/${meetingId}/action-items`);
  },
};
