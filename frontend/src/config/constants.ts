/** API_BASE_URL 保存模块使用的固定配置或共享状态。 */
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080';

/** MAX_UPLOAD_SIZE 保存模块使用的固定配置或共享状态。 */
export const MAX_UPLOAD_SIZE = 32 * 1024 * 1024;

/** statusOptions 保存模块使用的固定配置或共享状态。 */
export const statusOptions = ['在岗', '休假', '停用'];

/** shiftOptions 保存模块使用的固定配置或共享状态。 */
export const shiftOptions = ['白班', '夜班', '轮班', '弹性'];

/** menuStatusOptions 保存模块使用的固定配置或共享状态。 */
export const menuStatusOptions = ['启用', '停用'];

/** departmentStatusOptions 保存模块使用的固定配置或共享状态。 */
export const departmentStatusOptions = ['启用', '停用'];

/** roleStatusOptions 保存模块使用的固定配置或共享状态。 */
export const roleStatusOptions = ['启用', '停用'];

/** articleStatusOptions 保存模块使用的固定配置或共享状态。 */
export const articleStatusOptions = ['草稿', '已发布', '归档'];

/** pageKeys 保存模块使用的固定配置或共享状态。 */
export const pageKeys = ['dashboard', 'socket-support', 'visitor-analytics', 'users', 'departments', 'roles', 'menus', 'articles', 'files', 'profile'] as const;

/** pageTitles 定义对应业务的数据结构与调用契约。 */
export const pageTitles: Record<(typeof pageKeys)[number], string> = {
  dashboard: '预览台',
  'socket-support': '在线聊天',
  'visitor-analytics': '访问分析',
  users: '用户管理',
  departments: '部门管理',
  roles: '角色管理',
  menus: '菜单管理',
  articles: '文章管理',
  files: '文件管理',
  profile: '个人资料',
};

/** emptyUserForm 保存模块使用的固定配置或共享状态。 */
export const emptyUserForm = {
  username: '',
  name: '',
  role: '',
  roleId: null as number | null,
  department: '',
  departmentId: null as number | null,
  status: statusOptions[0],
  shift: shiftOptions[0],
  phone: '',
  email: '',
  canLogin: true,
  password: '',
};

/** emptyDepartmentForm 保存模块使用的固定配置或共享状态。 */
export const emptyDepartmentForm = {
  name: '',
  code: '',
  parentId: null as number | null,
  leader: '',
  phone: '',
  email: '',
  sort: 1,
  status: departmentStatusOptions[0],
};

/** emptyRoleForm 保存模块使用的固定配置或共享状态。 */
export const emptyRoleForm = {
  name: '',
  code: '',
  description: '',
  sort: 1,
  status: roleStatusOptions[0],
};

/** emptyMenuForm 保存模块使用的固定配置或共享状态。 */
export const emptyMenuForm = {
  name: '',
  code: '',
  path: '',
  icon: 'Menu',
  parentId: null as number | null,
  sort: 1,
  status: menuStatusOptions[0],
};

/** emptyArticleForm 保存模块使用的固定配置或共享状态。 */
export const emptyArticleForm = {
  title: '',
  category: '',
  author: '',
  status: articleStatusOptions[0],
  summary: '',
  content: '',
  isPrivate: false,
  portalVisible: false,
  portalFeatured: false,
  contentLocale: 'zh-CN',
};

/** emptyFileForm 保存模块使用的固定配置或共享状态。 */
export const emptyFileForm = {
  displayName: '',
  category: '',
  description: '',
  isPrivate: false,
  portalVisible: false,
  portalFeatured: false,
};
