let scrollLockedCount = 0

export function addScrollLockedCount(delta) {
  return (scrollLockedCount += delta)
}

export function getScrollLockedCount() {
  return scrollLockedCount
}
