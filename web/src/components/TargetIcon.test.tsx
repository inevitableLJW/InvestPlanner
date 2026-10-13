import { fireEvent, render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { TargetIcon } from './AppIcon'

describe('investment icon rendering', () => {
  it('recovers from a missing image when the selected target changes', () => {
    const { container, rerender } = render(<TargetIcon name="现金" />)
    const image = container.querySelector('img')!
    expect(image).toHaveAttribute('src', '/target-icons/cash.png')
    fireEvent.error(image)
    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('svg')).toBeInTheDocument()
    rerender(<TargetIcon name="债券类基金" />)
    expect(container.querySelector('img')).toHaveAttribute('src', '/target-icons/bond-fund.png')
  })

  it('keeps icons decorative so adjacent target names are announced only once', () => {
    const { container } = render(<TargetIcon name="纳指（QDII）" />)
    expect(container.querySelector('.target-icon')).toHaveAttribute('aria-hidden', 'true')
    expect(container.querySelector('img')).toHaveAttribute('alt', '')
  })
})
