export type ScenicSceneId = 'valley_trestle' | 'pine_forest';

export type ScreenId =
  | 'login'
  | 'admin_databases'
  | 'admin_users'
  | 'admin_groups'
  | 'admin_group_detail'
  | 'user_catalog'
  | 'db_detail'
  | 'draft_new'
  | 'draft_list'
  | 'draft_detail'
  | 'leader_approvals';

export interface AppNotification {
  id: string;
  title: string;
  body?: string;
  message?: string;
  time?: string;
  createdAtRelative?: string;
  unread: boolean;
  targetUrl?: string;
  targetScreen?: ScreenId;
  targetDraftId?: string;
  targetDbId?: string;
}

export interface SnackbarData {
  message: string;
  tone: 'pine' | 'maroon' | 'blue';
}
