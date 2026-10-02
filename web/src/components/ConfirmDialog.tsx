import { useCallback, useEffect, useState } from 'react'

export interface ConfirmOptions {
  title: string
  message: string
  confirmLabel?: string
  danger?: boolean
}

export function useConfirm() {
  const [opts, setOpts] = useState<ConfirmOptions | null>(null)
  const [resolve, setResolve] = useState<(v: boolean) => void>(() => () => {})

  const confirm = useCallback(
    (o: ConfirmOptions) => new Promise<boolean>(res => { setResolve(() => res); setOpts(o) }),
    []
  )

  const done = (v: boolean) => {
    setOpts(null)
    resolve(v)
  }

  useEffect(() => {
    if (!opts) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') done(false) }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [opts])

  const dialog = opts ? (
    <div className="overlay" onClick={e => { if (e.target === e.currentTarget) done(false) }}>
      <div className={`modal confirm-modal${opts.danger ? ' confirm-danger' : ''}`} role="alertdialog" aria-modal="true" aria-label={opts.title}>
        <h2>{opts.title}</h2>
        <p>{opts.message}</p>
        <div className="form-actions">
          <button className={opts.danger ? 'danger' : 'primary'} onClick={() => done(true)} autoFocus>
            {opts.confirmLabel ?? 'Confirm'}
          </button>
          <button className="ghost" onClick={() => done(false)}>Cancel</button>
        </div>
      </div>
    </div>
  ) : null

  return [dialog, confirm] as const
}
