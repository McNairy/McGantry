import { Component, type ErrorInfo, type ReactNode } from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('ErrorBoundary caught:', error, info.componentStack);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;
      return (
        <div className="flex min-h-[300px] flex-col items-center justify-center gap-4 rounded-xl border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-8 text-center">
          <AlertTriangle className="h-10 w-10 text-[var(--gantry-warning)]" />
          <div>
            <p className="text-base font-semibold text-[var(--gantry-text-primary)]">
              Something went wrong
            </p>
            <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">
              {this.state.error?.message ?? 'An unexpected error occurred.'}
            </p>
          </div>
          <button
            onClick={this.handleReset}
            className="flex items-center gap-2 rounded-lg border border-[var(--gantry-border)] px-4 py-2 text-sm text-[var(--gantry-text-primary)] hover:bg-[var(--gantry-bg-secondary)]"
          >
            <RefreshCw className="h-4 w-4" />
            Try again
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
