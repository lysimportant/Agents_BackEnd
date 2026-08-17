/** 图片画廊中单张图片的展示模型，由公开文件列表项转换而来。 */
export interface GalleryImage {
  /** 文件唯一标识。 */
  id: number;
  /** 浏览器可访问的绝对图片地址（完整预览）。 */
  src: string;
  /** 浏览器可访问的屏幕适配图片地址，用于瀑布流渐进加载。 */
  displaySrc?: string;
  /** 浏览器可访问的绝对缩略图地址，用于卡片省流量加载。 */
  thumbnailSrc?: string;
  /** 图片替代文本。 */
  alt: string;
  /** 图片宽度。 */
  width: number;
  /** 图片高度。 */
  height: number;
  /** 文件显示名称。 */
  displayName: string;
  /** 文件分类。 */
  category?: string;
  /** 文件描述。 */
  description?: string;
  /** 发布时间（RFC 3339）。 */
  publishedAt?: string;
}
