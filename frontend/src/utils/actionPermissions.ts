export type ActionPermissionDefinition = {
  code: string;
  label: string;
  description: string;
};

export type ActionPermissionGroup = {
  resource: string;
  label: string;
  actions: ActionPermissionDefinition[];
};

export type ResourceActionAccess = {
  create: boolean;
  update: boolean;
  delete: boolean;
  permissions?: boolean;
  restore?: boolean;
  permanentDelete?: boolean;
};

export const actionPermissionGroups: ActionPermissionGroup[] = [
  {
    resource: 'dashboard',
    label: '工作台',
    actions: [
      { code: 'dashboard.query', label: '查询', description: '加载工作台统计数据' },
      { code: 'dashboard.view', label: '查看', description: '查看工作台内容' },
      { code: 'dashboard.create', label: '新增', description: '新增采集数据' },
    ],
  },
  {
    resource: 'users',
    label: '用户管理',
    actions: [
      { code: 'users.query', label: '查询', description: '查询用户列表' },
      { code: 'users.view', label: '查看', description: '查看用户详情' },
      { code: 'users.create', label: '新增', description: '创建用户账号' },
      { code: 'users.update', label: '编辑', description: '修改用户资料' },
      { code: 'users.delete', label: '删除', description: '删除用户账号' },
      { code: 'users.permissions.update', label: '配置权限', description: '配置用户个人附加权限' },
    ],
  },
  {
    resource: 'departments',
    label: '部门管理',
    actions: [
      { code: 'departments.query', label: '查询', description: '查询部门列表' },
      { code: 'departments.view', label: '查看', description: '查看部门详情' },
      { code: 'departments.create', label: '新增', description: '创建部门' },
      { code: 'departments.update', label: '编辑', description: '修改部门资料' },
      { code: 'departments.delete', label: '删除', description: '删除部门' },
      { code: 'departments.permissions.update', label: '配置权限', description: '配置部门菜单权限' },
    ],
  },
  {
    resource: 'roles',
    label: '角色管理',
    actions: [
      { code: 'roles.query', label: '查询', description: '查询角色列表' },
      { code: 'roles.view', label: '查看', description: '查看角色详情' },
      { code: 'roles.create', label: '新增', description: '创建角色' },
      { code: 'roles.update', label: '编辑', description: '修改角色资料' },
      { code: 'roles.delete', label: '删除', description: '删除角色' },
      { code: 'roles.permissions.update', label: '配置权限', description: '配置角色菜单权限' },
    ],
  },
  {
    resource: 'menus',
    label: '菜单管理',
    actions: [
      { code: 'menus.query', label: '查询', description: '查询菜单列表' },
      { code: 'menus.view', label: '查看', description: '查看菜单详情' },
      { code: 'menus.create', label: '新增', description: '创建菜单' },
      { code: 'menus.update', label: '编辑', description: '修改菜单' },
      { code: 'menus.delete', label: '删除', description: '删除菜单' },
    ],
  },
  {
    resource: 'articles',
    label: '文章管理',
    actions: [
      { code: 'articles.query', label: '查询', description: '查询文章列表' },
      { code: 'articles.view', label: '查看', description: '查看文章内容' },
      { code: 'articles.create', label: '新增', description: '创建文章' },
      { code: 'articles.update', label: '编辑', description: '修改文章' },
      { code: 'articles.delete', label: '删除', description: '删除文章' },
    ],
  },
  {
    resource: 'files',
    label: '文件管理',
    actions: [
      { code: 'files.query', label: '查询', description: '查询文件列表' },
      { code: 'files.view', label: '查看', description: '预览和下载文件' },
      { code: 'files.create', label: '上传', description: '上传文件' },
      { code: 'files.update', label: '编辑', description: '修改文件资料' },
      { code: 'files.delete', label: '删除', description: '移入文件回收站' },
      { code: 'files.restore', label: '恢复', description: '从回收站恢复文件' },
      { code: 'files.permanent-delete', label: '永久删除', description: '永久删除回收站文件' },
    ],
  },
  {
    resource: 'socket',
    label: '在线聊天',
    actions: [
      { code: 'socket.query', label: '查询', description: '查询在线客户与历史会话' },
      { code: 'socket.view', label: '查看', description: '监视客户聊天内容' },
      { code: 'socket.send', label: '回复', description: '发送文字、图片、文件和表情' },
      { code: 'socket.delete', label: '删除', description: '软删除客服会话并从列表隐藏' },
    ],
  },
];

export const allActionPermissionCodes = actionPermissionGroups.flatMap((group) => group.actions.map((action) => action.code));
