import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { DashboardView } from './DashboardViews'

vi.mock('@/components/CodexMonitor', () => ({
  CodexMonitor: () => <div data-testid="codex-monitor" />,
}))

describe('DashboardView', () => {
  it('renders compact header without descriptive helper copy', () => {
    render(<DashboardView accessKey="secret" onLogout={() => {}} />)

    expect(screen.getByText('CLI Proxy API Dashboard')).toBeInTheDocument()
    expect(screen.queryByText('响应式管理面板，适配手机、平板与桌面端。')).not.toBeInTheDocument()
  })
})
