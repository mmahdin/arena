import { createContext, useContext, useLayoutEffect, useRef } from "react"
import { createPortal } from "react-dom"

// Native modal dialog supplies focus trapping, background inertness and Escape.
// This replaces the source app's Radix/Tailwind dialog without importing its UI stack.
const DialogContext = createContext(null)
export function Dialog({ open, onOpenChange, children }) {
  return open ? (
    <DialogContext.Provider value={onOpenChange}>
      {children}
    </DialogContext.Provider>
  ) : null
}
export function DialogContent({
  children,
  onOpenAutoFocus,
  onCloseAutoFocus,
  showCloseButton: _showCloseButton,
  ...props
}) {
  const ref = useRef(null)
  const onOpenChange = useContext(DialogContext)
  const callbacks = useRef({ onOpenAutoFocus, onCloseAutoFocus })
  callbacks.current = { onOpenAutoFocus, onCloseAutoFocus }
  useLayoutEffect(() => {
    const dialog = ref.current
    const previousFocus = document.activeElement
    const overflow = document.body.style.overflow
    document.body.style.overflow = "hidden"
    dialog.showModal()
    callbacks.current.onOpenAutoFocus?.({ preventDefault() {} })
    return () => {
      dialog.close()
      document.body.style.overflow = overflow
      if (callbacks.current.onCloseAutoFocus)
        callbacks.current.onCloseAutoFocus({ preventDefault() {} })
      else previousFocus?.focus()
    }
  }, [])
  return createPortal(
    <dialog
      {...props}
      ref={ref}
      onCancel={(event) => {
        event.preventDefault()
        onOpenChange(false)
      }}
      onKeyDown={(event) => {
        if (event.key !== "Tab") return
        const controls = [
          ...event.currentTarget.querySelectorAll(
            'button:not(:disabled), input:not(:disabled), [tabindex="0"]',
          ),
        ]
        const first = controls[0]
        const last = controls.at(-1)
        if (event.shiftKey && document.activeElement === first) {
          event.preventDefault()
          last?.focus()
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault()
          first?.focus()
        }
      }}
      onClick={(event) => {
        if (event.target !== event.currentTarget) return
        const rect = event.currentTarget.getBoundingClientRect()
        if (
          event.clientX < rect.left ||
          event.clientX > rect.right ||
          event.clientY < rect.top ||
          event.clientY > rect.bottom
        )
          onOpenChange(false)
      }}
    >
      {children}
    </dialog>,
    document.body,
  )
}
export function DialogTitle(props) {
  return <h2 {...props} />
}
export function DialogDescription(props) {
  return <p {...props} />
}
