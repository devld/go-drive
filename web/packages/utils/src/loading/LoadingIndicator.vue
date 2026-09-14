<template>
  <span class="loading-indicator" aria-hidden="true"></span>
</template>

<style lang="scss">
.loading-indicator {
  display: inline-block;
  box-sizing: border-box;
  width: 1em;
  height: 1em;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  vertical-align: -0.12em;
  animation: loading-indicator-spin 800ms linear infinite;
}

@keyframes loading-indicator-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .loading-indicator {
    animation: none;
  }
}
</style>
