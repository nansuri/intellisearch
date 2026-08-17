import { defineStore } from 'pinia'

export type Toast = { id: number; kind: 'success' | 'error' | 'info'; message: string }
let nextId = 1

export const useToastStore = defineStore('toast', {
  state: () => ({ toasts: [] as Toast[] }),
  actions: {
    push(kind: Toast['kind'], message: string) {
      const id = nextId++
      this.toasts.push({ id, kind, message })
      setTimeout(() => this.dismiss(id), 4200)
    },
    success(message: string) { this.push('success', message) },
    error(message: string) { this.push('error', message) },
    info(message: string) { this.push('info', message) },
    dismiss(id: number) { this.toasts = this.toasts.filter((t) => t.id !== id) },
  },
})