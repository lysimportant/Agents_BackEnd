import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

/** cn 实现对应业务逻辑。 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
