import { describe, expect, it, vi, beforeEach } from 'vitest'
import { resolveOpenListStreamUrl } from './stream'

const mockHttpClient = vi.fn()
const mockStreamUrl = vi.fn((id) => `/rest/stream?id=${id}`)

vi.mock('../dataProvider', () => ({
  httpClient: (...args) => mockHttpClient(...args),
}))

vi.mock('../subsonic', () => ({
  default: {
    streamUrl: (...args) => mockStreamUrl(...args),
  },
}))

vi.mock('../consts', () => ({
  REST_URL: '/api',
}))

describe('resolveOpenListStreamUrl', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns raw url when openlist resolve succeeds', async () => {
    mockHttpClient.mockResolvedValue({
      json: { rawUrl: 'https://openlist.local/d/Artist/Album/song.flac' },
    })

    const resolved = await resolveOpenListStreamUrl('song-1')

    expect(resolved).toBe('https://openlist.local/d/Artist/Album/song.flac')
    expect(mockHttpClient).toHaveBeenCalledWith('/api/openlist/stream/song-1')
  })

  it('falls back to subsonic stream url when id is empty', async () => {
    const resolved = await resolveOpenListStreamUrl('')

    expect(resolved).toBe('/rest/stream?id=')
    expect(mockHttpClient).not.toHaveBeenCalled()
  })

  it('falls back to provided stream url when openlist resolve fails', async () => {
    mockHttpClient.mockRejectedValue(new Error('network down'))

    const resolved = await resolveOpenListStreamUrl('song-1', '/fallback')

    expect(resolved).toBe('/fallback')
  })
})
