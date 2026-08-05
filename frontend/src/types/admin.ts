import type { CSSProperties } from 'react';

export type PageKey = 'dashboard' | 'socket-support' | 'visitor-analytics' | 'users' | 'departments' | 'roles' | 'menus' | 'articles' | 'files' | 'profile';

export type AuthUser = {
  id: number;
  username: string;
  name: string;
  role: string;
  roleId: number | null;
  roleCode: string;
  department: string;
  departmentId: number | null;
  status: string;
  phone: string;
  email: string;
  age: number;
  description: string;
  avatarUrl: string;
  canLogin: boolean;
  actionPermissions?: string[];
};

export type User = {
  id: number;
  username: string;
  name: string;
  role: string;
  roleId: number | null;
  roleCode: string;
  department: string;
  departmentId: number | null;
  status: string;
  shift: string;
  phone: string;
  email: string;
  age: number;
  description: string;
  avatarUrl: string;
  canLogin: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ProfileForm = {
  name: string;
  email: string;
  phone: string;
  age: number;
  description: string;
  avatarUrl: string;
};

export type Menu = {
  id: number;
  name: string;
  code: string;
  path: string;
  icon: string;
  parentId: number | null;
  sort: number;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type MenuNode = Menu & {
  depth: number;
  children: MenuNode[];
};

export type Department = {
  id: number;
  name: string;
  code: string;
  parentId: number | null;
  leader: string;
  phone: string;
  email: string;
  sort: number;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type DepartmentForm = {
  name: string;
  code: string;
  parentId: number | null;
  leader: string;
  phone: string;
  email: string;
  sort: number;
  status: string;
};

export type Role = {
  id: number;
  name: string;
  code: string;
  description: string;
  sort: number;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type RoleForm = {
  name: string;
  code: string;
  description: string;
  sort: number;
  status: string;
};

export type UserPermissionDetails = {
  departmentMenuIds: number[];
  roleMenuIds: number[];
  userMenuIds: number[];
  effectiveMenuIds: number[];
  roleActionCodes: string[];
  userActionCodes: string[];
  effectiveActionCodes: string[];
};

export type Article = {
  id: number;
  title: string;
  category: string;
  author: string;
  status: string;
  summary: string;
  content: string;
  views: number;
  ownerId?: number;
  ownerName?: string;
  isPrivate?: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ManagedFile = {
  id: number;
  source?: 'internal-chat' | 'customer-chat';
  displayName: string;
  originalName: string;
  category: string;
  description: string;
  contentType: string;
  size: number;
  storageName: string;
  ownerId?: number;
  ownerName?: string;
  isPrivate?: boolean;
  readOnly?: boolean;
  previewUrl?: string;
  downloadUrl?: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string | null;
};

export type LoginForm = {
  username: string;
  password: string;
};

export type UserForm = {
  username: string;
  name: string;
  role: string;
  roleId: number | null;
  department: string;
  departmentId: number | null;
  status: string;
  shift: string;
  phone: string;
  email: string;
  canLogin: boolean;
  password: string;
};

export type MenuForm = {
  name: string;
  code: string;
  path: string;
  icon: string;
  parentId: number | null;
  sort: number;
  status: string;
};

export type ArticleForm = {
  title: string;
  category: string;
  author: string;
  status: string;
  summary: string;
  content: string;
  isPrivate: boolean;
};

export type FileForm = {
  displayName: string;
  category: string;
  description: string;
  isPrivate: boolean;
};

export type DepthStyle = CSSProperties & {
  '--depth'?: number;
};

export type VisitorAnalyticsRange = '24h' | '7d' | '30d';

export type VisitorAccessRecord = {
  id: number;
  ip: string;
  forwardedIp?: string;
  country: string;
  region: string;
  city: string;
  isp: string;
  host: string;
  method: string;
  path: string;
  statusCode: number;
  durationMs: number;
  bytes: number;
  userAgent: string;
  browser: string;
  os: string;
  device: string;
  referer: string;
  acceptLanguage: string;
  userId?: number;
  userName?: string;
  authenticated: boolean;
  createdAt: string;
};

export type VisitorAnalyticsDimension = {
  name: string;
  value: number;
};

export type VisitorAnalyticsPoint = {
  label: string;
  value: number;
};

export type VisitorAnalyticsResponse = {
  records: VisitorAccessRecord[];
  total: number;
  page: number;
  pageSize: number;
  summary: {
    totalRequests: number;
    uniqueIps: number;
    authenticatedRequests: number;
    errorRequests: number;
    averageDurationMs: number;
    countries: VisitorAnalyticsDimension[];
    paths: VisitorAnalyticsDimension[];
    timeline: VisitorAnalyticsPoint[];
  };
};
