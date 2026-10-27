export type Dict<T = unknown> = Record<string, T>;

export interface ApiEnvelope<T> {
  code: number;
  message: string;
  data?: T;
  request_id?: string;
}

export interface PaginationQuery {
  page?: number;
  page_size?: number;
}

export interface Pagination {
  page: number;
  page_size: number;
  total: number;
}

export interface PageResult<T> {
  list: T[];
  pagination: Pagination;
}

export interface PersonRef extends Dict {
  user_id?: string;
  name: string;
}

export interface EntityRef extends Dict {
  entity_id: string;
  entity_type: string;
}

export interface FileRef extends Dict {
  file_id: string;
  file_name: string;
  file_url: string;
  file_size: number;
}

export interface UserInfo extends Dict {
  user_id: string;
  username: string;
  real_name: string;
  email: string;
  roles: string[];
  permissions: string[];
  departments: string[];
}

export interface LoginRequest {
  username: string;
  password: string;
  captcha?: string;
  captcha_key?: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token?: string;
  token_type: "Bearer" | string;
  expires_in: number;
  user_info?: UserInfo;
}

export interface RefreshTokenRequest {
  refresh_token: string;
}

export interface RegisterRequest extends Dict {
  username: string;
  password: string;
  confirm_password: string;
  email: string;
  real_name: string;
}

export interface RegisterResponse extends Dict {
  user_id: string;
  username: string;
  email: string;
}

export interface PasswordChangeRequest {
  old_password: string;
  new_password: string;
  confirm_password: string;
}

export interface ForgotPasswordRequest {
  account: string;
}

export interface ForgotPasswordResponse extends Dict {
  reset_token?: string;
  expires_in: number;
  message: string;
}

export interface ResetPasswordRequest {
  reset_token: string;
  new_password: string;
  confirm_password: string;
}

export interface AgentSummary extends Dict {
  agent_id: string;
  agent_name: string;
}
export interface AgentDetail extends AgentSummary {}
export interface AgentListQuery extends PaginationQuery, Dict {}
export interface AgentInvokeRequest extends Dict {}
export interface AgentInvokeResponse extends Dict {
  task_id: string;
  status: string;
}

export interface TodoQuery extends PaginationQuery, Dict {}
export interface TodoItem extends Dict {
  todo_id: string;
  title: string;
  status: string;
}
export interface CreateTodoRequest extends Dict {
  title: string;
  category: string;
  priority: string;
  due_date: string;
}
export interface UpdateTodoStatusRequest extends Dict {
  status: string;
}
export interface NotificationRequest extends Dict {
  title: string;
  content: string;
  type: string;
  recipients: string[];
}

export interface KnowledgeSearchQuery extends Dict {
  query: string;
  company_code: string;
}
export interface KnowledgeSearchResult extends Dict {
  results: Dict[];
  total: number;
}

export interface EmailQuery extends PaginationQuery, Dict {}
export interface EmailItem extends Dict {
  email_id: string;
  subject: string;
}
export interface SendEmailRequest extends Dict {
  to: string[];
  subject: string;
  body: string;
}

export interface MeetingMinutesRequest extends Dict {
  participants: string[];
}
export interface MeetingMinutesResponse extends Dict {
  minutes_id: string;
  summary: string;
}

export interface LPQuery extends PaginationQuery, Dict {}
export interface LPItem extends Dict {
  lp_id: string;
  lp_name: string;
}
export interface CreateLPRequest extends Dict {
  lp_name: string;
  lp_type: string;
}

export interface FundQuery extends PaginationQuery, Dict {}
export interface FundItem extends Dict {
  fund_id: string;
  fund_name: string;
}
export interface CreateFundRequest extends Dict {
  fund_name: string;
  fund_type: string;
  target_size: number;
}
export interface ComplianceCheckRequest extends Dict {
  subject_id: string;
  subject_type: string;
  check_types: string[];
}

export interface InvestmentProjectQuery extends PaginationQuery, Dict {}
export interface InvestmentProjectItem extends Dict {
  project_id: string;
  project_name: string;
}
export interface CreateInvestmentProjectRequest extends Dict {
  project_name: string;
  industry: string;
}
export interface DCFValuationRequest extends Dict {
  revenue: number;
  wacc: number;
}
export interface DCFValuationResponse extends Dict {
  valuation_id: string;
  project_id: string;
}

export interface MeetingSummary extends Dict {
  overview?: string;
  key_points?: Array<Dict | string>;
  action_items?: MeetingActionItem[];
}
export interface CreateMeetingRequest extends Dict {
  project_id: string;
  meeting_title: string;
  agenda?: string;
  meeting_date?: string;
  participants?: string[];
}
export interface CreateMeetingResponse {
  meeting_id: string;
  status: string;
}
export interface MeetingDetail extends Dict {
  meeting_id: string;
  meeting_title: string;
  agenda?: string;
  meeting_date?: string;
  participants?: string[];
  audio_url?: string;
  duration?: number;
  raw_text?: string;
  ai_summary?: MeetingSummary;
  status?: string;
}
export interface UploadMeetingAudioResponse extends Dict {
  audio_id: string;
  file_size: number;
  duration: number;
  format: string;
}
export interface TranscribeMeetingRequest {
  enable_correction?: boolean;
  asr_backend?: string;
}
export interface TranscribeMeetingResponse extends Dict {
  transcript_id: string;
  raw_text: string;
  corrected_text?: string;
  ai_summary: MeetingSummary;
  processing_time: number;
  asr_backend: string;
  has_correction: boolean;
}
export interface MeetingActionItem extends Dict {
  item: string;
  owner: string;
  deadline: string;
  synchronized: boolean;
}
export interface MeetingActionItemsResponse {
  items: MeetingActionItem[];
}

export interface ExitOpportunityQuery extends PaginationQuery, Dict {}
export interface ExitOpportunityItem extends Dict {
  opportunity_id: string;
  project_id: string;
}
export interface ExitAnalysisRequest extends Dict {
  analysis_type: string;
  time_horizon: string;
}

export interface InitUploadRequest {
  file_name: string;
  file_size: number;
  chunk_count: number;
}
export interface InitUploadResponse extends Dict {
  upload_id: string;
  chunk_size: number;
  expires_at: string;
  upload_url: string;
}
export interface ChunkUploadResponse {
  upload_id: string;
  chunk_index: number;
  received: boolean;
}
export interface CompleteUploadResponse extends Dict {
  file_id: string;
  file_url: string;
  file_size: number;
  md5: string;
  status: string;
}
