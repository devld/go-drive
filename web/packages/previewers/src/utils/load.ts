export async function loadBlob(url: string, signal?: AbortSignal) {
  const response = await fetch(url, { signal })
  if (!response.ok) {
    throw new Error(
      `Unable to load preview file (${response.status} ${response.statusText})`
    )
  }
  return response.blob()
}

export async function loadArrayBuffer(url: string, signal?: AbortSignal) {
  return (await loadBlob(url, signal)).arrayBuffer()
}
