/** 列表页查询参数的解析结果。 */
export interface ListSearchParams {
  /** 页码，从 1 开始。 */
  page: number;
  /** 分类名。 */
  category: string;
  /** 关键词。 */
  keyword: string;
  /** 排序方式。 */
  sort: string;
}

/** 读取 Next.js searchParams 中可能为数组或单值的参数。 */
function readParam(value: string | string[] | undefined): string {
  return Array.isArray(value) ? (value[0] ?? '') : (value ?? '');
}

/**
 * 将列表页 searchParams 解析为类型安全的查询参数，非法页码回退到第 1 页。
 */
export function parseListSearchParams(
  params: Record<string, string | string[] | undefined>,
): ListSearchParams {
  const rawPage = Number.parseInt(readParam(params.page), 10);
  const page = Number.isFinite(rawPage) && rawPage >= 1 ? rawPage : 1;
  return {
    page,
    category: readParam(params.category),
    keyword: readParam(params.keyword),
    sort: readParam(params.sort),
  };
}
