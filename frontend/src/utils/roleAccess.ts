/** SUPER_ADMIN_ROLE_CODE 保存模块使用的固定配置或共享状态。 */
export const SUPER_ADMIN_ROLE_CODE = 'super-admin';

/** SYSTEM_ADMIN_ROLE_CODE 保存模块使用的固定配置或共享状态。 */
export const SYSTEM_ADMIN_ROLE_CODE = 'system-admin';

/** isSuperAdminRoleCode 校验对应业务条件。 */
export function isSuperAdminRoleCode(roleCode?: string | null) {
  return roleCode === SUPER_ADMIN_ROLE_CODE;
}

/** isAdministratorRoleCode 校验对应业务条件。 */
export function isAdministratorRoleCode(roleCode?: string | null) {
  return roleCode === SUPER_ADMIN_ROLE_CODE || roleCode === SYSTEM_ADMIN_ROLE_CODE;
}
