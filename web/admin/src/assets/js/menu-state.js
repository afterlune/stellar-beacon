let currentMenuPromise = null
let loaded = false

export function isMenuLoaded() {
  return loaded
}

export function getMenuPromise() {
  return currentMenuPromise
}

export function setMenuPromise(promise) {
  currentMenuPromise = promise
}

export function markMenuLoaded() {
  loaded = true
}

export function resetMenuState() {
  currentMenuPromise = null
  loaded = false
}
