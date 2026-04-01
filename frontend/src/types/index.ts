export interface User {
  id:         number;
  name:       string;
  email:      string;
  avatar_url?: string;
  created_at: string;
  updated_at?: string;
}

export interface Folder {
  id:         number;
  user_id:    number;
  parent_id:  number | null;
  name:       string;
  trashed:    boolean;
  created_at: string;
  updated_at: string;
}

export interface FileItem {
  id:          number;
  user_id:     number;
  folder_id:   number | null;
  file_name:   string;
  file_type:   string;
  file_size:   number;
  // 'processing' = chunks received, S3 upload in progress (async)
  status:      'pending' | 'uploading' | 'processing' | 'completed' | 'failed';
  starred:     boolean;
  trashed:     boolean;
  created_at:  string;
  updated_at:  string;
}

export interface BreadcrumbItem {
  id:   number | null;
  name: string;
}