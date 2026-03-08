import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { RequestLogDialog } from './RequestLogDialog'
import type { RequestLogRecord } from '../types'

const failedLog: RequestLogRecord = {
  id: 1,
  timestamp: '2026-03-07T14:00:00.000Z',
  firstTokenMs: 120,
  durationMs: 320,
  totalTokens: 0,
  cacheReadTokens: 0,
  cacheWriteTokens: 0,
  statusCode: 401,
  success: false,
  model: 'gpt-5.4',
  errorMessage: '{"error":"Invalid API key"}',
  responseBody: '{"error":"Invalid API key"}',
}

describe('RequestLogDialog', () => {
  it('shows failure title by status code instead of raw error text', () => {
    render(<RequestLogDialog log={failedLog} onClose={() => {}} />)

    expect(screen.getByRole('heading', { name: '失败(401)' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: '{"error":"Invalid API key"}' })).not.toBeInTheDocument()
  })

  it('does not render a duplicate error info paragraph when response body already contains details', () => {
    render(<RequestLogDialog log={failedLog} onClose={() => {}} />)

    expect(screen.queryByText('错误信息：{"error":"Invalid API key"}')).not.toBeInTheDocument()
  })

  it('uses a natural chinese label for the response body section', () => {
    render(<RequestLogDialog log={failedLog} onClose={() => {}} />)

    expect(screen.getByRole('heading', { name: '响应内容' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'responseBody' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '复制响应内容' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '复制 responseBody' })).not.toBeInTheDocument()
  })

  it('shows first token and total duration separately', () => {
    render(<RequestLogDialog log={failedLog} onClose={() => {}} />)

    expect(screen.getByText('性能 120ms / 320ms / --')).toBeInTheDocument()
  })
})
