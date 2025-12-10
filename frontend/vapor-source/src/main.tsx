import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.tsx'
import globalStyles from './index.css?inline'

function injectGlobalStyles() {
  const existing = document.head.querySelector('style[data-inline-css=\"app\"]')
  if (existing) return
  const style = document.createElement('style')
  style.setAttribute('data-inline-css', 'app')
  style.textContent = globalStyles
  document.head.appendChild(style)
}

injectGlobalStyles()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
