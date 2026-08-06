import type { Menu, MenuNode } from '@/src/types/admin';

/** buildMenuTree 转换并生成对应业务结果。 */
export function buildMenuTree(menus: Menu[]) {
  /** sortedMenus 负责计算或维护排序结果菜单。 */
  const sortedMenus = [...menus].sort((first, second) => first.sort - second.sort || first.id - second.id);
  /** appendChildren 保存子节点。 */
  const appendChildren = (parentId: number | null, depth: number): MenuNode[] =>
    sortedMenus
      .filter((menu) => (menu.parentId ?? null) === parentId)
      .flatMap((menu) => [{ ...menu, depth, children: [] }, ...appendChildren(menu.id, depth + 1)]);

  return appendChildren(null, 0);
}
