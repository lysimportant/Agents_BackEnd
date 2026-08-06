import * as React from "react"

/** MOBILE_BREAKPOINT 保存模块使用的固定配置或共享状态。 */
const MOBILE_BREAKPOINT = 768

/** useIsMobile 实现对应业务逻辑。 */
export function useIsMobile() {
  /** isMobile、setIsMobile 保存移动端、移动端。 */
  const [isMobile, setIsMobile] = React.useState<boolean | undefined>(undefined)

  React.useEffect(() => {
    /** mql 保存变量 mql。 */
    const mql = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`)
    /** onChange 负责处理对应的界面事件和状态变化。 */
    const onChange = () => {
      setIsMobile(window.innerWidth < MOBILE_BREAKPOINT)
    }
    mql.addEventListener("change", onChange)
    setIsMobile(window.innerWidth < MOBILE_BREAKPOINT)
    return () => mql.removeEventListener("change", onChange)
  }, [])

  return !!isMobile
}
