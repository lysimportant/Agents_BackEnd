import type { Menu, MenuNode } from '../types/admin';

export function buildMenuTree(menus: Menu[]) {
  const sortedMenus = [...menus].sort((first, second) => first.sort - second.sort || first.id - second.id);
  const appendChildren = (parentId: number | null, depth: number): MenuNode[] =>
    sortedMenus
      .filter((menu) => menu.parentId === parentId)
      .flatMap((menu) => [{ ...menu, depth, children: [] }, ...appendChildren(menu.id, depth + 1)]);

  return appendChildren(null, 0);
}
