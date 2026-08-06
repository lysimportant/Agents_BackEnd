import type { ReactNode } from 'react';

type DataTableProps = {
  /** children 表示子节点。 */
  children: ReactNode;
};

/** DataTable 实现对应业务逻辑。 */
export function DataTable({ children }: DataTableProps) {
  return <table>{children}</table>;
}
