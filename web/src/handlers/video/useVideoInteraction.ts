import { computed, onMounted, onUnmounted, ref, type Ref } from 'vue'

const CONTROL_HIDE_DELAY = 3000
const DOUBLE_TAP_DELAY = 300
const DOUBLE_TAP_DISTANCE = 32
const SWIPE_START_DISTANCE = 8
const SWIPE_DISMISS_DISTANCE_RATIO = 0.2
const SWIPE_FLICK_DISTANCE = 56
const SWIPE_FLICK_VELOCITY = 0.8
const SWIPE_SETTLE_DURATION = 120

type SwipeDirection = 'pending' | 'down' | 'other'

interface SwipeGesture {
  pointerId: number
  startX: number
  startY: number
  startAt: number
  direction: SwipeDirection
}

export const useVideoInteraction = (
  containerEl: Ref<HTMLElement | undefined>,
  videoEl: Ref<HTMLVideoElement | undefined>,
  hasPlaybackStarted: Ref<boolean>,
  togglePlay: () => void,
  close: () => void
) => {
  const controlsHidden = ref(false)
  const usesTouchControls = ref(false)
  const swipeOffsetY = ref(0)
  const swipeOpacity = ref(1)
  const isSwipeDragging = ref(false)
  const isDismissing = ref(false)

  let hideTimer = 0
  let touchTapTimer = 0
  let dismissTimer = 0
  let lastTouchTapAt = 0
  let lastTouchTapX = 0
  let lastTouchTapY = 0
  let initialTapStartedPlayback = false
  let swipeGesture: SwipeGesture | undefined

  const swipeStyle = computed(() => ({
    transform:
      swipeOffsetY.value > 0
        ? `translate3d(0, ${swipeOffsetY.value}px, 0)`
        : undefined,
    opacity: swipeOpacity.value,
  }))

  const scheduleHide = () => {
    clearTimeout(hideTimer)
    hideTimer = window.setTimeout(() => {
      controlsHidden.value = true
    }, CONTROL_HIDE_DELAY)
  }

  const toggleTouchControls = () => {
    if (!usesTouchControls.value) return
    clearTimeout(hideTimer)
    controlsHidden.value = !controlsHidden.value
    if (!controlsHidden.value) scheduleHide()
  }

  const resetSwipe = () => {
    swipeGesture = undefined
    isSwipeDragging.value = false
    swipeOffsetY.value = 0
    swipeOpacity.value = 1
  }

  const onPlayerPointerDown = (event: PointerEvent) => {
    if (!usesTouchControls.value || event.pointerType !== 'touch') return
    if (isDismissing.value) return

    if (!event.isPrimary) {
      if (swipeGesture) {
        swipeGesture.direction = 'other'
        isSwipeDragging.value = false
        swipeOffsetY.value = 0
        swipeOpacity.value = 1
      }
      return
    }

    const target = event.currentTarget
    if (target instanceof HTMLElement) {
      target.setPointerCapture(event.pointerId)
    }

    swipeGesture = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      startAt: Date.now(),
      direction: 'pending',
    }
  }

  const updateSwipe = (event: PointerEvent) => {
    const gesture = swipeGesture
    if (!gesture || gesture.pointerId !== event.pointerId) return

    const deltaX = event.clientX - gesture.startX
    const deltaY = event.clientY - gesture.startY
    if (
      gesture.direction === 'pending' &&
      Math.hypot(deltaX, deltaY) >= SWIPE_START_DISTANCE
    ) {
      gesture.direction =
        deltaY > 0 && deltaY >= Math.abs(deltaX) * 1.2 ? 'down' : 'other'
    }

    if (gesture.direction === 'down') {
      const height = containerEl.value?.clientHeight || window.innerHeight
      const offset = Math.max(0, deltaY)
      isSwipeDragging.value = true
      swipeOffsetY.value = offset
      swipeOpacity.value = Math.max(0, 1 - offset / (height * 0.5))
    }

    return gesture
  }

  const finishPendingTap = () => {
    if (touchTapTimer === 0) return
    clearTimeout(touchTapTimer)
    touchTapTimer = 0
    lastTouchTapAt = 0
    initialTapStartedPlayback = false
    toggleTouchControls()
  }

  const finishSwipe = (event: PointerEvent) => {
    const gesture = updateSwipe(event)
    if (!gesture || gesture.direction === 'pending') {
      swipeGesture = undefined
      return false
    }

    finishPendingTap()

    const deltaY = Math.max(0, event.clientY - gesture.startY)
    const elapsed = Math.max(1, Date.now() - gesture.startAt)
    const height = containerEl.value?.clientHeight || window.innerHeight
    const shouldDismiss =
      gesture.direction === 'down' &&
      (deltaY >= height * SWIPE_DISMISS_DISTANCE_RATIO ||
        (deltaY >= SWIPE_FLICK_DISTANCE &&
          deltaY / elapsed >= SWIPE_FLICK_VELOCITY))

    if (shouldDismiss) {
      swipeGesture = undefined
      isSwipeDragging.value = false
      isDismissing.value = true
      swipeOffsetY.value = Math.max(window.innerHeight, deltaY)
      swipeOpacity.value = 0

      const delay = window.matchMedia('(prefers-reduced-motion: reduce)').matches
        ? 0
        : SWIPE_SETTLE_DURATION
      dismissTimer = window.setTimeout(close, delay)
      return true
    }

    resetSwipe()
    return true
  }

  const onPlayerPointerMove = (event: PointerEvent) => {
    if (event.pointerType === 'touch') updateSwipe(event)
  }

  const onPlayerPointerCancel = (event: PointerEvent) => {
    if (swipeGesture?.pointerId === event.pointerId) resetSwipe()
  }

  const onPlayerPointerUp = (event: PointerEvent) => {
    if (!usesTouchControls.value || event.pointerType !== 'touch') return
    if (!event.isPrimary || isDismissing.value) return
    if (finishSwipe(event)) return

    const now = Date.now()
    const isDoubleTap =
      touchTapTimer !== 0 &&
      now - lastTouchTapAt <= DOUBLE_TAP_DELAY &&
      Math.hypot(event.clientX - lastTouchTapX, event.clientY - lastTouchTapY) <=
        DOUBLE_TAP_DISTANCE

    if (isDoubleTap) {
      clearTimeout(touchTapTimer)
      touchTapTimer = 0
      lastTouchTapAt = 0

      // Preserve the first tap's autoplay gesture instead of immediately
      // undoing the playback it starts when a double tap is detected.
      if (initialTapStartedPlayback) {
        if (!hasPlaybackStarted.value && videoEl.value?.paused) togglePlay()
      } else {
        togglePlay()
      }
      initialTapStartedPlayback = false
      if (!controlsHidden.value) scheduleHide()
      return
    }

    // Finish a nearby single tap before starting another one that is not a
    // double tap.
    if (touchTapTimer !== 0) {
      clearTimeout(touchTapTimer)
      touchTapTimer = 0
      toggleTouchControls()
    }

    lastTouchTapAt = now
    lastTouchTapX = event.clientX
    lastTouchTapY = event.clientY

    if (!hasPlaybackStarted.value && videoEl.value?.paused) {
      togglePlay()
      initialTapStartedPlayback = true
    } else {
      initialTapStartedPlayback = false
    }

    touchTapTimer = window.setTimeout(() => {
      touchTapTimer = 0
      lastTouchTapAt = 0
      initialTapStartedPlayback = false
      toggleTouchControls()
    }, DOUBLE_TAP_DELAY)
  }

  const onDesktopPlayerClick = () => {
    if (!usesTouchControls.value) togglePlay()
  }

  const onControlsClick = (event: MouseEvent) => {
    if (!usesTouchControls.value) return
    const target = event.target
    if (
      target instanceof Element &&
      target.closest(
        '.video-controls__btn, .video-controls__bar, .video-controls__volume-bar'
      )
    ) {
      scheduleHide()
      return
    }
    clearTimeout(hideTimer)
    controlsHidden.value = true
  }

  const onControlsPointerActivity = () => {
    if (!controlsHidden.value) scheduleHide()
  }

  const showControls = () => {
    if (usesTouchControls.value) return
    controlsHidden.value = false
    scheduleHide()
  }

  onMounted(() => {
    usesTouchControls.value = window.matchMedia(
      '(hover: none) and (pointer: coarse)'
    ).matches
    scheduleHide()
  })

  onUnmounted(() => {
    clearTimeout(hideTimer)
    clearTimeout(touchTapTimer)
    clearTimeout(dismissTimer)
  })

  return {
    controlsHidden,
    usesTouchControls,
    onPageClick: toggleTouchControls,
    onPlayerPointerDown,
    onPlayerPointerMove,
    onPlayerPointerCancel,
    onPlayerPointerUp,
    onPlayerClick: onDesktopPlayerClick,
    onPlayerDoubleClick: onDesktopPlayerClick,
    onControlsClick,
    onControlsPointerActivity,
    onPlayerKeyboardActivate: togglePlay,
    showControls,
    scheduleHide,
    swipeStyle,
    isSwipeDragging,
    isDismissing,
  }
}
