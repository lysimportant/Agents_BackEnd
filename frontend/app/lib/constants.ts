export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080';

export const MAX_UPLOAD_SIZE = 32 * 1024 * 1024;

export const roleOptions = ['系统管理员', '内容编辑', '数据分析师', '值班员'];
export const statusOptions = ['在岗', '休假', '停用'];
export const shiftOptions = ['白班', '夜班', '轮班', '弹性'];
export const menuStatusOptions = ['启用', '停用'];
export const articleStatusOptions = ['草稿', '已发布', '归档'];
export const pageKeys = ['dashboard', 'users', 'menus', 'articles', 'files'] as const;

export const pageTitles: Record<(typeof pageKeys)[number], string> = {
  dashboard: '工作台',
  users: '用户管理',
  menus: '菜单管理',
  articles: '文章管理',
  files: '文件管理',
};

export const emptyUserForm = {
  username: '',
  name: '',
  role: roleOptions[0],
  department: '',
  status: statusOptions[0],
  shift: shiftOptions[0],
  phone: '',
  email: '',
  canLogin: true,
  password: '',
};

export const emptyMenuForm = {
  name: '',
  code: '',
  path: '',
  icon: 'Menu',
  parentId: null as number | null,
  sort: 1,
  status: menuStatusOptions[0],
};

export const emptyArticleForm = {
  title: '',
  category: '',
  author: '',
  status: articleStatusOptions[0],
  summary: '',
  content: '',
  isPrivate: false,
};

export const emptyFileForm = {
  displayName: '',
  category: '',
  description: '',
  isPrivate: false,
};
