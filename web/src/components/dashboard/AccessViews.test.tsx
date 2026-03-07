import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LoginCard } from './AccessViews'

describe('LoginCard', () => {
  it('renders login form without descriptive helper copy', () => {
    render(
      <LoginCard
        draftKey=""
        error=""
        loading={false}
        onChange={() => {}}
        onSubmit={() => {}}
      />,
    )

    expect(screen.getByText('CLI Proxy API Dashboard')).toBeInTheDocument()
    expect(screen.queryByText('输入 Management Key 后进入响应式 Dashboard。')).not.toBeInTheDocument()
  })
})
