let scrollLockedCount = 0

export function addScrollLockedCount(delta: number) {
  return (scrollLockedCount += delta)
}

export function getScrollLockedCount() {
  return scrollLockedCount
}
