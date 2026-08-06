import type { CSSProperties } from 'react';

/** 使用不受可编辑菜单名称影响的稳定编码标识工作台页面。 */
export type PageKey = 'dashboard' | 'socket-support' | 'visitor-analytics' | 'users' | 'departments' | 'roles' | 'menus' | 'articles' | 'files' | 'profile';

/** 会话 API 可安全返回的已登录用户数据。 */
export type AuthUser = {
  /** id 表示标识。 */
  id: number;
  /** username 表示用户名。 */
  username: string;
  /** name 表示名称。 */
  name: string;
  /** role 表示角色。 */
  role: string;
  /** roleId 表示角色标识。 */
  roleId: number | null;
  /** roleCode 表示角色编码。 */
  roleCode: string;
  /** department 表示部门。 */
  department: string;
  /** departmentId 表示部门标识。 */
  departmentId: number | null;
  /** status 表示状态。 */
  status: string;
  /** phone 表示电话号码。 */
  phone: string;
  /** email 表示邮箱地址。 */
  email: string;
  /** age 表示年龄。 */
  age: number;
  /** description 表示说明。 */
  description: string;
  /** avatarUrl 表示头像地址。 */
  avatarUrl: string;
  /** canLogin 表示登录。 */
  canLogin: boolean;
  /** actionPermissions 表示操作权限权限。 */
  actionPermissions?: string[];
};

/** 包含角色和部门投影信息的管理端用户记录。 */
export type User = {
  /** id 表示标识。 */
  id: number;
  /** username 表示用户名。 */
  username: string;
  /** name 表示名称。 */
  name: string;
  /** role 表示角色。 */
  role: string;
  /** roleId 表示角色标识。 */
  roleId: number | null;
  /** roleCode 表示角色编码。 */
  roleCode: string;
  /** department 表示部门。 */
  department: string;
  /** departmentId 表示部门标识。 */
  departmentId: number | null;
  /** status 表示状态。 */
  status: string;
  /** shift 表示班次。 */
  shift: string;
  /** phone 表示电话号码。 */
  phone: string;
  /** email 表示邮箱地址。 */
  email: string;
  /** age 表示年龄。 */
  age: number;
  /** description 表示说明。 */
  description: string;
  /** avatarUrl 表示头像地址。 */
  avatarUrl: string;
  /** canLogin 表示登录。 */
  canLogin: boolean;
  /** createdAt 表示创建时间。 */
  createdAt: string;
  /** updatedAt 表示更新时间。 */
  updatedAt: string;
};

/** 当前用户资料表单中允许编辑的字段。 */
export type ProfileForm = {
  /** name 表示名称。 */
  name: string;
  /** email 表示邮箱地址。 */
  email: string;
  /** phone 表示电话号码。 */
  phone: string;
  /** age 表示年龄。 */
  age: number;
  /** description 表示说明。 */
  description: string;
  /** avatarUrl 表示头像地址。 */
  avatarUrl: string;
};

/** 用于导航和鉴权的菜单记录。 */
export type Menu = {
  /** id 表示标识。 */
  id: number;
  /** name 表示名称。 */
  name: string;
  /** code 表示编码。 */
  code: string;
  /** path 表示路径。 */
  path: string;
  /** icon 表示图标。 */
  icon: string;
  /** parentId 表示标识。 */
  parentId: number | null;
  /** sort 表示排序。 */
  sort: number;
  /** status 表示状态。 */
  status: string;
  /** createdAt 表示创建时间。 */
  createdAt: string;
  /** updatedAt 表示更新时间。 */
  updatedAt: string;
};

/** 附加树深度和子节点、用于渲染的菜单记录。 */
export type MenuNode = Menu & {
  depth: number;
  children: MenuNode[];
};

/** 组织部门记录。 */
export type Department = {
  /** id 表示标识。 */
  id: number;
  /** name 表示名称。 */
  name: string;
  /** code 表示编码。 */
  code: string;
  /** parentId 表示标识。 */
  parentId: number | null;
  /** leader 表示负责人。 */
  leader: string;
  /** phone 表示电话号码。 */
  phone: string;
  /** email 表示邮箱地址。 */
  email: string;
  /** sort 表示排序。 */
  sort: number;
  /** status 表示状态。 */
  status: string;
  /** createdAt 表示创建时间。 */
  createdAt: string;
  /** updatedAt 表示更新时间。 */
  updatedAt: string;
};

/** 部门表单的可编辑字段值。 */
export type DepartmentForm = {
  /** name 表示名称。 */
  name: string;
  /** code 表示编码。 */
  code: string;
  /** parentId 表示标识。 */
  parentId: number | null;
  /** leader 表示负责人。 */
  leader: string;
  /** phone 表示电话号码。 */
  phone: string;
  /** email 表示邮箱地址。 */
  email: string;
  /** sort 表示排序。 */
  sort: number;
  /** status 表示状态。 */
  status: string;
};

/** 由稳定权限编码标识的角色记录。 */
export type Role = {
  /** id 表示标识。 */
  id: number;
  /** name 表示名称。 */
  name: string;
  /** code 表示编码。 */
  code: string;
  /** description 表示说明。 */
  description: string;
  /** sort 表示排序。 */
  sort: number;
  /** status 表示状态。 */
  status: string;
  /** createdAt 表示创建时间。 */
  createdAt: string;
  /** updatedAt 表示更新时间。 */
  updatedAt: string;
};

/** 角色表单的可编辑字段值。 */
export type RoleForm = {
  /** name 表示名称。 */
  name: string;
  /** code 表示编码。 */
  code: string;
  /** description 表示说明。 */
  description: string;
  /** sort 表示排序。 */
  sort: number;
  /** status 表示状态。 */
  status: string;
};

/** 指定用户的菜单和动作权限来源及其有效并集。 */
export type UserPermissionDetails = {
  /** departmentMenuIds 表示部门菜单标识列表。 */
  departmentMenuIds: number[];
  /** roleMenuIds 表示角色菜单标识列表。 */
  roleMenuIds: number[];
  /** userMenuIds 表示用户菜单标识列表。 */
  userMenuIds: number[];
  /** effectiveMenuIds 表示最终生效菜单标识列表。 */
  effectiveMenuIds: number[];
  /** roleActionCodes 表示角色。 */
  roleActionCodes: string[];
  /** userActionCodes 表示用户。 */
  userActionCodes: string[];
  /** effectiveActionCodes 表示最终生效。 */
  effectiveActionCodes: string[];
};

/** 包含所有权和隐私元数据的知识库文章。 */
export type Article = {
  /** id 表示标识。 */
  id: number;
  /** title 表示标题。 */
  title: string;
  /** category 表示分类。 */
  category: string;
  /** author 表示作者。 */
  author: string;
  /** status 表示状态。 */
  status: string;
  /** summary 表示摘要。 */
  summary: string;
  /** content 表示内容。 */
  content: string;
  /** views 表示查看次数。 */
  views: number;
  /** ownerId 表示所有者标识。 */
  ownerId?: number;
  /** ownerName 表示所有者名称。 */
  ownerName?: string;
  /** isPrivate 表示私密状态。 */
  isPrivate?: boolean;
  /** createdAt 表示创建时间。 */
  createdAt: string;
  /** updatedAt 表示更新时间。 */
  updatedAt: string;
};

/** 文件管理记录，包含受保护聊天数据的投影视图。 */
export type ManagedFile = {
  /** id 表示标识。 */
  id: number;
  /** source 表示来源。 */
  source?: 'internal-chat' | 'customer-chat';
  /** displayName 表示名称。 */
  displayName: string;
  /** originalName 表示名称。 */
  originalName: string;
  /** category 表示分类。 */
  category: string;
  /** description 表示说明。 */
  description: string;
  /** contentType 表示内容。 */
  contentType: string;
  /** size 表示大小。 */
  size: number;
  /** storageName 表示存储名称。 */
  storageName: string;
  /** ownerId 表示所有者标识。 */
  ownerId?: number;
  /** ownerName 表示所有者名称。 */
  ownerName?: string;
  /** isPrivate 表示私密状态。 */
  isPrivate?: boolean;
  /** readOnly 表示只读状态。 */
  readOnly?: boolean;
  /** previewUrl 表示预览地址。 */
  previewUrl?: string;
  /** downloadUrl 表示地址。 */
  downloadUrl?: string;
  /** createdAt 表示创建时间。 */
  createdAt: string;
  /** updatedAt 表示更新时间。 */
  updatedAt: string;
  /** deletedAt 表示删除状态。 */
  deletedAt?: string | null;
};

/** 登录表单采集的凭据。 */
export type LoginForm = {
  /** username 表示用户名。 */
  username: string;
  /** password 表示密码。 */
  password: string;
};

/** 管理员用户表单的可编辑字段值。 */
export type UserForm = {
  /** username 表示用户名。 */
  username: string;
  /** name 表示名称。 */
  name: string;
  /** role 表示角色。 */
  role: string;
  /** roleId 表示角色标识。 */
  roleId: number | null;
  /** department 表示部门。 */
  department: string;
  /** departmentId 表示部门标识。 */
  departmentId: number | null;
  /** status 表示状态。 */
  status: string;
  /** shift 表示班次。 */
  shift: string;
  /** phone 表示电话号码。 */
  phone: string;
  /** email 表示邮箱地址。 */
  email: string;
  /** canLogin 表示登录。 */
  canLogin: boolean;
  /** password 表示密码。 */
  password: string;
};

/** 菜单表单的可编辑字段值。 */
export type MenuForm = {
  /** name 表示名称。 */
  name: string;
  /** code 表示编码。 */
  code: string;
  /** path 表示路径。 */
  path: string;
  /** icon 表示图标。 */
  icon: string;
  /** parentId 表示标识。 */
  parentId: number | null;
  /** sort 表示排序。 */
  sort: number;
  /** status 表示状态。 */
  status: string;
};

/** 文章表单的可编辑字段值。 */
export type ArticleForm = {
  /** title 表示标题。 */
  title: string;
  /** category 表示分类。 */
  category: string;
  /** author 表示作者。 */
  author: string;
  /** status 表示状态。 */
  status: string;
  /** summary 表示摘要。 */
  summary: string;
  /** content 表示内容。 */
  content: string;
  /** isPrivate 表示私密状态。 */
  isPrivate: boolean;
};

/** 受管文件元数据表单的可编辑字段值。 */
export type FileForm = {
  /** displayName 表示名称。 */
  displayName: string;
  /** category 表示分类。 */
  category: string;
  /** description 表示说明。 */
  description: string;
  /** isPrivate 表示私密状态。 */
  isPrivate: boolean;
};

/** 根据层级缩进菜单树行的 CSS 自定义属性。 */
export type DepthStyle = CSSProperties & {
  '--depth'?: number;
};

/** 访问分析查询支持的滚动时间范围。 */
export type VisitorAnalyticsRange = '24h' | '7d' | '30d';

/** 单次 HTTP 请求采集的隐私敏感访问元数据。 */
export type VisitorAccessRecord = {
  /** id 表示标识。 */
  id: number;
  /** ip 表示变量 ip。 */
  ip: string;
  /** forwardedIp 表示变量 forwardedIp。 */
  forwardedIp?: string;
  /** country 表示国家或地区。 */
  country: string;
  /** region 表示变量 region。 */
  region: string;
  /** city 表示变量 city。 */
  city: string;
  /** isp 表示变量 isp。 */
  isp: string;
  /** host 表示主机地址。 */
  host: string;
  /** method 表示请求方法。 */
  method: string;
  /** path 表示路径。 */
  path: string;
  /** statusCode 表示状态编码。 */
  statusCode: number;
  /** durationMs 表示耗时。 */
  durationMs: number;
  /** bytes 表示字节数。 */
  bytes: number;
  /** userAgent 表示用户。 */
  userAgent: string;
  /** browser 表示浏览器。 */
  browser: string;
  /** os 表示变量 os。 */
  os: string;
  /** device 表示设备。 */
  device: string;
  /** referer 表示变量 referer。 */
  referer: string;
  /** acceptLanguage 表示变量 acceptLanguage。 */
  acceptLanguage: string;
  /** userId 表示用户标识。 */
  userId?: number;
  /** userName 表示用户名称。 */
  userName?: string;
  /** authenticated 表示变量 authenticated。 */
  authenticated: boolean;
  /** createdAt 表示创建时间。 */
  createdAt: string;
};

/** 访问分析图表展示的具名聚合维度。 */
export type VisitorAnalyticsDimension = {
  /** name 表示名称。 */
  name: string;
  /** value 表示值。 */
  value: number;
};

/** 访问分析时间序列中的一个带标签数据点。 */
export type VisitorAnalyticsPoint = {
  /** label 表示显示标签。 */
  label: string;
  /** value 表示值。 */
  value: number;
};

/** 访问分析 API 返回的分页记录和聚合数据。 */
export type VisitorAnalyticsResponse = {
  /** records 表示记录。 */
  records: VisitorAccessRecord[];
  /** total 表示总数。 */
  total: number;
  /** page 表示页码。 */
  page: number;
  /** pageSize 表示页码大小。 */
  pageSize: number;
  /** summary 表示摘要。 */
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
