type Handler = (data: unknown) => void;

class EventBus {
  private handlers: Map<string, Set<Handler>> = new Map();

  on(event: string, handler: Handler): () => void {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set());
    }
    this.handlers.get(event)!.add(handler);
    return () => {
      this.handlers.get(event)?.delete(handler);
    };
  }

  once(event: string, handler: Handler): () => void {
    const wrapper: Handler = (data) => {
      unsub();
      handler(data);
    };
    const unsub = this.on(event, wrapper);
    return unsub;
  }

  emit(event: string, data?: unknown): void {
    this.handlers.get(event)?.forEach(h => {
      try { h(data); } catch (_e) { /* swallow handler errors */ }
    });
  }

  off(event: string, handler?: Handler): void {
    if (handler) {
      this.handlers.get(event)?.delete(handler);
    } else {
      this.handlers.delete(event);
    }
  }

  clear(): void {
    this.handlers.clear();
  }
}

export const eventBus = new EventBus();
