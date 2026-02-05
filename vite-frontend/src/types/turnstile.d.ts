declare global {
  interface Window {
    turnstile?: {
      render: (container: HTMLElement | string, options: Record<string, any>) => any;
      reset: (widgetId?: any) => void;
      remove: (widgetId: any) => void;
    };
  }
}

export {};
