import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../dataProvider', () => ({
  httpClient: vi.fn(),
}))

import { httpClient } from '../dataProvider'
import subsonic from './index'

describe('resolveOpenListStreamUrl', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns raw url when openlist resolve succeeds', async () => {
    httpClient.mockResolvedValue({
      json: { rawUrl: 'https://openlist.local/d/Artist/Album/song.flac' },
    })

    const resolved = await subsonic.resolveOpenListStreamUrl(
      'song-1',
      '/rest/stream?id=song-1',
    )

    expect(resolved).toBe('https://openlist.local/d/Artist/Album/song.flac')
    expect(httpClient).toHaveBeenCalledWith('/api/openlist/stream/song-1')
  })

  it('falls back to stream url when raw url is empty', async () => {
    httpClient.mockResolvedValue({
      json: { rawUrl: '' },
    })

    const fallback = '/rest/stream?id=song-1'
    const resolved = await subsonic.resolveOpenListStreamUrl('song-1', fallback)

    expect(resolved).toBe(fallback)
  })

  it('falls back to stream url when openlist resolve fails', async () => {
    httpClient.mockRejectedValue(new Error('network down'))

    const fallback = '/rest/stream?id=song-1'
    const resolved = await subsonic.resolveOpenListStreamUrl('song-1', fallback)

    expect(resolved).toBe(fallback)
  })
})
