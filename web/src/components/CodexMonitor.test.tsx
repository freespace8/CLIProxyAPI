import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { CodexMonitor } from './CodexMonitor'
import type { RequestLogStreamEvent } from '../types'
import { openCodexRequestLogStream } from '../api'

vi.mock('../api', async () => {
  const actual = await vi.importActual<typeof import('../api')>('../api')
  return {
    ...actual,
    openCodexRequestLogStream: vi.fn(),
  }
})

const openCodexRequestLogStreamMock = vi.mocked(openCodexRequestLogStream)

function streamFromEvents(events: RequestLogStreamEvent[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  return new ReadableStream<Uint8Array>({
    start(controller) {
      for (const event of events) {
        controller.enqueue(encoder.encode(`${JSON.stringify(event)}\n`))
      }
      controller.close()
    },
  })
}

afterEach(() => {
  vi.clearAllMocks()
})

describe('CodexMonitor responsive layout', () => {
  it('does not render explanatory marketing copy in the dashboard panels', async () => {
    openCodexRequestLogStreamMock.mockResolvedValueOnce(streamFromEvents([
      {
        type: 'snapshot',
        requests: [],
        logs: [],
      },
    ]))

    render(<CodexMonitor accessKey="secret" />)

    expect(await screen.findByText('请求监控')).toBeInTheDocument()
    expect(screen.queryByText('覆盖手机、平板和桌面端的实时请求与日志视图。')).not.toBeInTheDocument()
    expect(screen.queryByText('小屏单列展示，平板与桌面自动扩展为多列卡片。')).not.toBeInTheDocument()
    expect(screen.queryByText('移动端使用摘要卡片，桌面端保留完整表格。')).not.toBeInTheDocument()
  })

  it('keeps live request cards on the original compact content layout', async () => {
    openCodexRequestLogStreamMock.mockResolvedValueOnce(streamFromEvents([
      {
        type: 'snapshot',
        requests: [{
          requestId: 'req-1',
          model: 'gpt-5-codex',
          thinkingLevel: 'high',
          serviceTier: 'priority',
          startTime: '2026-03-07T14:00:00.000Z',
        }],
        logs: [],
      },
    ]))

    render(<CodexMonitor accessKey="secret" />)

    expect(await screen.findByText('gpt-5-codex high fast')).toBeInTheDocument()
    expect(screen.queryByText('开始时间')).not.toBeInTheDocument()
    expect(screen.queryByText('已运行')).not.toBeInTheDocument()
    expect(screen.queryByText('请求正在处理中，已切换为移动端友好的摘要布局。')).not.toBeInTheDocument()
  })

  it('renders recent logs as mobile cards while keeping desktop table headings', async () => {
    openCodexRequestLogStreamMock.mockResolvedValueOnce(streamFromEvents([
      {
        type: 'snapshot',
        requests: [],
        logs: [{
          id: 42,
          timestamp: '2026-03-07T14:00:00.000Z',
          firstTokenMs: 240,
          durationMs: 1820,
          outputTokens: 512,
          totalTokens: 1536,
          cacheReadTokens: 256,
          cacheWriteTokens: 64,
          statusCode: 500,
          success: false,
          model: 'gpt-5-codex',
          thinkingLevel: 'medium',
          serviceTier: 'priority',
          errorMessage: 'upstream timeout',
          responseBody: '{"error":"timeout"}',
        }],
      },
    ]))

    render(<CodexMonitor accessKey="secret" />)

    expect((await screen.findAllByText('gpt-5-codex medium fast')).length).toBeGreaterThan(0)

    await waitFor(() => {
      expect(screen.getAllByText('性能').length).toBeGreaterThan(0)
      expect(screen.getByText('总 Token')).toBeInTheDocument()
      expect(screen.getByText('缓存读取')).toBeInTheDocument()
      expect(screen.getByText('缓存写入')).toBeInTheDocument()
    })

    expect(screen.getAllByText('失败(500)').length).toBeGreaterThan(0)
    expect(screen.getAllByText('--').length).toBeGreaterThan(0)
    expect(screen.queryByText('upstream timeout')).not.toBeInTheDocument()
    expect(screen.getByText('读缓存')).toBeInTheDocument()
    expect(screen.getByText('写缓存')).toBeInTheDocument()
  })

  it('keeps live request cards compact without reintroducing table overflow containers', async () => {
    openCodexRequestLogStreamMock.mockResolvedValueOnce(streamFromEvents([
      {
        type: 'snapshot',
        requests: [{
          requestId: 'req-2',
          model: 'gpt-5.4',
          startTime: '2026-03-07T14:00:00.000Z',
        }],
        logs: [{
          id: 99,
          timestamp: '2026-03-07T14:00:00.000Z',
          firstTokenMs: 260,
          durationMs: 920,
          outputTokens: 330,
          totalTokens: 640,
          cacheReadTokens: 128,
          cacheWriteTokens: 32,
          statusCode: 200,
          success: true,
          model: 'gpt-5.4',
          responseBody: '',
        }],
      },
    ]))

    render(<CodexMonitor accessKey="secret" />)

    expect((await screen.findAllByText('gpt-5.4')).length).toBeGreaterThan(0)
    expect(screen.getAllByText('260ms/920ms/500tok/s').length).toBeGreaterThan(0)

    const liveRequestsGrid = screen.getByTestId('live-requests-grid')
    expect(liveRequestsGrid.className).toContain('flex')
    expect(liveRequestsGrid.className).toContain('flex-wrap')

    const desktopTable = screen.getByTestId('logs-desktop-table')
    const desktopGrid = screen.getByTestId('logs-desktop-grid')

    expect(desktopTable.className).not.toContain('overflow-x-auto')
    expect(desktopGrid.className).not.toContain('min-w-[940px]')
  })

  it('computes tok/s from output tokens and generation duration', async () => {
    openCodexRequestLogStreamMock.mockResolvedValueOnce(streamFromEvents([
      {
        type: 'snapshot',
        requests: [],
        logs: [{
          id: 101,
          timestamp: '2026-03-07T14:00:00.000Z',
          firstTokenMs: 1110,
          durationMs: 4580,
          outputTokens: 37,
          totalTokens: 181,
          cacheReadTokens: 113540,
          cacheWriteTokens: 0,
          statusCode: 200,
          success: true,
          model: 'gpt-5.4',
          responseBody: '',
        }],
      },
    ]))

    render(<CodexMonitor accessKey="secret" />)

    expect((await screen.findAllByText('1.11s/4.58s/11tok/s')).length).toBeGreaterThan(0)
  })
})
