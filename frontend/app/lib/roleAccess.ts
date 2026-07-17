export const SUPER_ADMIN_ROLE_CODE = 'super-admin';
export const SYSTEM_ADMIN_ROLE_CODE = 'system-admin';

export function isSuperAdminRoleCode(roleCode?: string | null) {
  return roleCode === SUPER_ADMIN_ROLE_CODE;
}

export function isAdministratorRoleCode(roleCode?: string | null) {
  return roleCode === SUPER_ADMIN_ROLE_CODE || roleCode === SYSTEM_ADMIN_ROLE_CODE;
}
