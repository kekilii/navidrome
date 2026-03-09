import React from 'react'
import { cleanup, render, waitFor } from '@testing-library/react'
import { Player } from './Player'
import subsonic from '../subsonic'

let latestPlayerProps = null
let mockState = null
const mockDispatch = vi.fn()
const mockDataProvider = { getOne: vi.fn(() => Promise.resolve({})) }
const sentAudioSources = []

vi.mock('react-redux', () => ({
  useDispatch: () => mockDispatch,
  useSelector: (selector) => selector(mockState),
}))

vi.mock('@material-ui/core', async () => {
  const actual = await import('@material-ui/core')
  return {
    ...actual,
    useMediaQuery: () => true,
  }
})

vi.mock('react-admin', () => ({
  createMuiTheme: () => ({}),
  useAuthState: () => ({ authenticated: true }),
  useDataProvider: () => mockDataProvider,
  useTranslate: () => (key) => key,
}))

vi.mock('navidrome-music-player', () => ({
  default: (props) => {
    latestPlayerProps = props
    return <div data-testid="mock-player" />
  },
}))

vi.mock('../themes/useCurrentTheme', () => ({
  default: () => ({ player: { theme: 'dark' } }),
}))

vi.mock('./styles', () => ({
  default: () => ({ player: 'mock-player-class' }),
}))

vi.mock('./AudioTitle', () => ({
  default: () => null,
}))

vi.mock('./PlayerToolbar', () => ({
  default: () => null,
}))

vi.mock('../utils', () => ({
  sendNotification: vi.fn(),
}))

vi.mock('../hotkeys', () => ({
  keyMap: {},
}))

vi.mock('./keyHandlers', () => ({
  default: () => ({}),
}))

vi.mock('../utils/calculateReplayGain', () => ({
  calculateGain: () => 1,
}))

vi.mock('../subsonic', () => ({
  default: {
    resolveOpenListStreamUrl: vi.fn(),
    streamUrl: vi.fn((id) => `/rest/stream?id=${id}`),
    scrobble: vi.fn(),
    nowPlaying: vi.fn(),
  },
}))

class MockAudio {
  set src(value) {
    this._src = value
    sentAudioSources.push(value)
  }

  get src() {
    return this._src
  }
}

const makeTrack = (id, uuid = `uuid-${id}`) => ({
  uuid,
  trackId: id,
  musicSrc: `/rest/stream?id=${id}`,
  song: {
    title: `Song ${id}`,
    artist: 'Artist',
    album: 'Album',
    albumId: '1',
  },
})

const makeState = () => ({
  player: {
    queue: [makeTrack('song-1', 'uuid-1'), makeTrack('song-2', 'uuid-2')],
    current: {
      uuid: 'uuid-1',
      trackId: 'song-1',
      song: { title: 'Song 1', artist: 'Artist', album: 'Album', albumId: '1' },
    },
    clear: false,
    volume: 0.8,
    mode: 'order',
    playIndex: 0,
  },
  settings: { notifications: false },
  replayGain: { gainMode: 'off' },
})

describe('<Player /> OpenList preload', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    latestPlayerProps = null
    sentAudioSources.length = 0
    mockState = makeState()
    global.Audio = MockAudio
  })

  afterEach(() => {
    cleanup()
  })

  it('uses openlist resolver during half-progress preload', async () => {
    subsonic.resolveOpenListStreamUrl.mockResolvedValue(
      'https://openlist/raw-song-2',
    )

    render(<Player />)

    latestPlayerProps.onAudioProgress({
      ended: false,
      currentTime: 200,
      duration: 300,
      isRadio: false,
      trackId: 'song-1',
    })

    await waitFor(() => {
      expect(subsonic.resolveOpenListStreamUrl).toHaveBeenCalledWith(
        'song-2',
        '/rest/stream?id=song-2',
      )
    })
    await waitFor(() => {
      expect(sentAudioSources).toContain('https://openlist/raw-song-2')
    })
  })

  it('reuses the same resolved url without duplicate resolve call', async () => {
    subsonic.resolveOpenListStreamUrl.mockResolvedValue(
      'https://openlist/raw-song-2',
    )

    render(<Player />)

    const nextTrackSrc = latestPlayerProps.audioLists[1].musicSrc
    expect(typeof nextTrackSrc).toBe('function')

    latestPlayerProps.onAudioProgress({
      ended: false,
      currentTime: 200,
      duration: 300,
      isRadio: false,
      trackId: 'song-1',
    })

    await waitFor(() => {
      expect(subsonic.resolveOpenListStreamUrl).toHaveBeenCalledTimes(1)
    })

    const secondResolved = await nextTrackSrc()
    expect(secondResolved).toBe('https://openlist/raw-song-2')
    expect(subsonic.resolveOpenListStreamUrl).toHaveBeenCalledTimes(1)
  })

  it('does not retry openlist for the same track after fallback', async () => {
    subsonic.resolveOpenListStreamUrl.mockRejectedValue(new Error('network down'))

    render(<Player />)

    const song2Src = latestPlayerProps.audioLists[1].musicSrc
    expect(typeof song2Src).toBe('function')

    const firstResolved = await song2Src()
    expect(firstResolved).toBe('/rest/stream?id=song-2')
    expect(subsonic.resolveOpenListStreamUrl).toHaveBeenCalledTimes(1)

    const secondResolved = await song2Src()
    expect(secondResolved).toBe('/rest/stream?id=song-2')
    expect(subsonic.resolveOpenListStreamUrl).toHaveBeenCalledTimes(1)
  })

  it('still tries openlist for a different track after one fallback', async () => {
    subsonic.resolveOpenListStreamUrl
      .mockRejectedValueOnce(new Error('song-2 down'))
      .mockResolvedValueOnce('https://openlist/raw-song-3')

    const { rerender } = render(<Player />)

    const song2Src = latestPlayerProps.audioLists[1].musicSrc
    await song2Src()
    expect(subsonic.resolveOpenListStreamUrl).toHaveBeenCalledTimes(1)
    expect(subsonic.resolveOpenListStreamUrl).toHaveBeenNthCalledWith(
      1,
      'song-2',
      '/rest/stream?id=song-2',
    )

    mockState = {
      ...mockState,
      player: {
        ...mockState.player,
        queue: [
          makeTrack('song-1', 'uuid-1'),
          makeTrack('song-2', 'uuid-2'),
          makeTrack('song-3', 'uuid-3'),
        ],
      },
    }
    rerender(<Player />)

    const song3Src = latestPlayerProps.audioLists[2].musicSrc
    expect(typeof song3Src).toBe('function')
    const song3Resolved = await song3Src()

    expect(song3Resolved).toBe('https://openlist/raw-song-3')
    expect(subsonic.resolveOpenListStreamUrl).toHaveBeenCalledTimes(2)
    expect(subsonic.resolveOpenListStreamUrl).toHaveBeenNthCalledWith(
      2,
      'song-3',
      '/rest/stream?id=song-3',
    )
  })

  it('keeps fallback decision for a track removed and re-added in the same session', async () => {
    subsonic.resolveOpenListStreamUrl.mockRejectedValue(new Error('network down'))

    const { rerender } = render(<Player />)

    const song2Src = latestPlayerProps.audioLists[1].musicSrc
    await song2Src()
    expect(subsonic.resolveOpenListStreamUrl).toHaveBeenCalledTimes(1)

    mockState = {
      ...mockState,
      player: {
        ...mockState.player,
        queue: [makeTrack('song-1', 'uuid-1b')],
      },
    }
    rerender(<Player />)

    mockState = {
      ...mockState,
      player: {
        ...mockState.player,
        queue: [makeTrack('song-1', 'uuid-1b'), makeTrack('song-2', 'uuid-2b')],
      },
    }
    rerender(<Player />)

    const readdedSong2Src = latestPlayerProps.audioLists[1].musicSrc
    if (typeof readdedSong2Src === 'function') {
      const resolved = await readdedSong2Src()
      expect(resolved).toBe('/rest/stream?id=song-2')
    } else {
      expect(readdedSong2Src).toBe('/rest/stream?id=song-2')
    }
    expect(subsonic.resolveOpenListStreamUrl).toHaveBeenCalledTimes(1)
  })
})
