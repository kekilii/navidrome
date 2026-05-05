import { httpClient } from '../dataProvider'
import { REST_URL } from '../consts'
import subsonic from '../subsonic'

export const resolveOpenListStreamUrl = async (id, fallbackUrl = '') => {
  const fallback = fallbackUrl || subsonic.streamUrl(id)
  if (!id) {
    return fallback
  }
  try {
    const resp = await httpClient(
      `${REST_URL}/openlist/stream/${encodeURIComponent(id)}`,
    )
    const rawUrl = resp?.json?.rawUrl?.trim?.() ?? ''
    if (rawUrl) {
      return rawUrl
    }
  } catch (_e) {
    // Silent fallback to the default stream endpoint.
  }
  return fallback
}
